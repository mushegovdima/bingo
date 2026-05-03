package notificationworker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	notificationcontract "go.mod/internal/contracts/notification"
	dbmodels "go.mod/internal/db"
	"go.mod/internal/db/repository"
	"go.mod/internal/notifier"
)

// --- helpers ---

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func mustFilter(t *testing.T, f notificationcontract.UserFilter) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal filter: %v", err)
	}
	return b
}

func newJob(id int64, filter json.RawMessage, attempts int) *dbmodels.NotificationJob {
	job := &dbmodels.NotificationJob{
		Type:     "season_available",
		Text:     "hello",
		Filter:   filter,
		Status:   string(notificationcontract.JobStatusRunning),
		Attempts: attempts,
	}
	job.ID = id
	return job
}

// --- fakes ---

type fakeJobRepo struct {
	mu sync.Mutex

	claimFn        func(ctx context.Context, limit int) ([]*dbmodels.NotificationJob, error)
	updateCursorFn func(ctx context.Context, id, cursor int64) error
	markDoneFn     func(ctx context.Context, id int64) error
	markFailedFn   func(ctx context.Context, id int64, attempts int, errMsg string, terminal bool) error

	updateCursorCalls []cursorCall
	doneIDs           []int64
	failedCalls       []failedCall
}

type cursorCall struct {
	id, cursor int64
}

type failedCall struct {
	id       int64
	attempts int
	errMsg   string
	terminal bool
}

func (f *fakeJobRepo) Claim(ctx context.Context, limit int) ([]*dbmodels.NotificationJob, error) {
	if f.claimFn != nil {
		return f.claimFn(ctx, limit)
	}
	return nil, nil
}

func (f *fakeJobRepo) UpdateCursor(ctx context.Context, id, cursor int64) error {
	f.mu.Lock()
	f.updateCursorCalls = append(f.updateCursorCalls, cursorCall{id, cursor})
	f.mu.Unlock()
	if f.updateCursorFn != nil {
		return f.updateCursorFn(ctx, id, cursor)
	}
	return nil
}

func (f *fakeJobRepo) MarkDone(ctx context.Context, id int64) error {
	f.mu.Lock()
	f.doneIDs = append(f.doneIDs, id)
	f.mu.Unlock()
	if f.markDoneFn != nil {
		return f.markDoneFn(ctx, id)
	}
	return nil
}

func (f *fakeJobRepo) MarkFailed(ctx context.Context, id int64, attempts int, errMsg string, terminal bool) error {
	f.mu.Lock()
	f.failedCalls = append(f.failedCalls, failedCall{id, attempts, errMsg, terminal})
	f.mu.Unlock()
	if f.markFailedFn != nil {
		return f.markFailedFn(ctx, id, attempts, errMsg, terminal)
	}
	return nil
}

type fakeUserRepo struct {
	listFn      func(ctx context.Context, filter repository.UserFilter, afterID int64, limit int) ([]*dbmodels.User, error)
	listCalls   []listCall
	mu          sync.Mutex
	pages       map[int64][]*dbmodels.User // afterID -> page (returned then "exhausted")
	lastFilter  repository.UserFilter
	filterCalls int
}

type listCall struct {
	afterID int64
	limit   int
}

func (f *fakeUserRepo) ListByFilter(ctx context.Context, filter repository.UserFilter, afterID int64, limit int) ([]*dbmodels.User, error) {
	f.mu.Lock()
	f.listCalls = append(f.listCalls, listCall{afterID, limit})
	f.lastFilter = filter
	f.filterCalls++
	f.mu.Unlock()

	if f.listFn != nil {
		return f.listFn(ctx, filter, afterID, limit)
	}
	if f.pages != nil {
		return f.pages[afterID], nil
	}
	return nil, nil
}

type fakeSender struct {
	mu       sync.Mutex
	sendBulk func(ctx context.Context, notificationType string, recipients []notifier.BulkRecipient) error
	calls    []bulkCall
}

type bulkCall struct {
	notifType  string
	recipients []notifier.BulkRecipient
}

func (f *fakeSender) Send(ctx context.Context, userID, telegramID int64, text string) error {
	return nil
}

func (f *fakeSender) SendBulk(ctx context.Context, notificationType string, recipients []notifier.BulkRecipient) error {
	f.mu.Lock()
	f.calls = append(f.calls, bulkCall{notificationType, append([]notifier.BulkRecipient(nil), recipients...)})
	f.mu.Unlock()
	if f.sendBulk != nil {
		return f.sendBulk(ctx, notificationType, recipients)
	}
	return nil
}

func makeUsers(ids ...int64) []*dbmodels.User {
	out := make([]*dbmodels.User, len(ids))
	for i, id := range ids {
		u := &dbmodels.User{TelegramID: 100 + id}
		u.ID = id
		out[i] = u
	}
	return out
}

func newWorker(jobs *fakeJobRepo, users *fakeUserRepo, sender *fakeSender, cfg Config) *Worker {
	return New(jobs, users, sender, cfg, noopLogger())
}

// --- New defaults ---

func TestNew_FillsDefaults(t *testing.T) {
	w := newWorker(&fakeJobRepo{}, &fakeUserRepo{}, &fakeSender{}, Config{})

	if w.cfg.PollInterval != defaultPollInterval {
		t.Errorf("PollInterval: want %v, got %v", defaultPollInterval, w.cfg.PollInterval)
	}
	if w.cfg.BatchSize != defaultBatchSize {
		t.Errorf("BatchSize: want %d, got %d", defaultBatchSize, w.cfg.BatchSize)
	}
	if w.cfg.MaxAttempts != defaultMaxAttempts {
		t.Errorf("MaxAttempts: want %d, got %d", defaultMaxAttempts, w.cfg.MaxAttempts)
	}
}

func TestNew_ClampsBatchSizeOverMax(t *testing.T) {
	w := newWorker(&fakeJobRepo{}, &fakeUserRepo{}, &fakeSender{},
		Config{BatchSize: notifier.MaxBulkRecipients + 1})

	if w.cfg.BatchSize != defaultBatchSize {
		t.Errorf("expected oversized BatchSize to fall back to default %d, got %d",
			defaultBatchSize, w.cfg.BatchSize)
	}
}

func TestNew_KeepsExplicitValues(t *testing.T) {
	cfg := Config{
		PollInterval: 17 * time.Second,
		BatchSize:    7,
		JobBatch:     3,
		MaxAttempts:  9,
	}
	w := newWorker(&fakeJobRepo{}, &fakeUserRepo{}, &fakeSender{}, cfg)

	if w.cfg != cfg {
		t.Errorf("config mutated: want %+v, got %+v", cfg, w.cfg)
	}
}

// --- tick ---

func TestTick_EmptyClaim(t *testing.T) {
	jobs := &fakeJobRepo{
		claimFn: func(_ context.Context, _ int) ([]*dbmodels.NotificationJob, error) { return nil, nil },
	}
	w := newWorker(jobs, &fakeUserRepo{}, &fakeSender{}, Config{JobBatch: 4})

	processed, err := w.tick(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if processed {
		t.Error("expected processed=false on empty queue")
	}
}

func TestTick_ClaimError(t *testing.T) {
	boom := errors.New("db down")
	jobs := &fakeJobRepo{
		claimFn: func(_ context.Context, _ int) ([]*dbmodels.NotificationJob, error) { return nil, boom },
	}
	w := newWorker(jobs, &fakeUserRepo{}, &fakeSender{}, Config{JobBatch: 2})

	processed, err := w.tick(context.Background())

	if !errors.Is(err, boom) {
		t.Fatalf("want boom, got %v", err)
	}
	if processed {
		t.Error("expected processed=false on claim error")
	}
}

func TestTick_PassesJobBatchToClaim(t *testing.T) {
	var seen int32
	jobs := &fakeJobRepo{
		claimFn: func(_ context.Context, limit int) ([]*dbmodels.NotificationJob, error) {
			atomic.StoreInt32(&seen, int32(limit))
			return nil, nil
		},
	}
	w := newWorker(jobs, &fakeUserRepo{}, &fakeSender{}, Config{JobBatch: 7})

	if _, err := w.tick(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atomic.LoadInt32(&seen); got != 7 {
		t.Errorf("Claim limit: want 7, got %d", got)
	}
}

func TestTick_ProcessesAllClaimedJobs(t *testing.T) {
	filter := mustFilter(t, notificationcontract.UserFilter{})
	job1 := newJob(1, filter, 0)
	job2 := newJob(2, filter, 0)

	jobs := &fakeJobRepo{
		claimFn: func(_ context.Context, _ int) ([]*dbmodels.NotificationJob, error) {
			return []*dbmodels.NotificationJob{job1, job2}, nil
		},
	}
	users := &fakeUserRepo{} // returns nil, nil → empty audience
	sender := &fakeSender{}

	w := newWorker(jobs, users, sender, Config{JobBatch: 4, BatchSize: 10})

	processed, err := w.tick(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !processed {
		t.Error("expected processed=true")
	}
	if len(jobs.doneIDs) != 2 || jobs.doneIDs[0] != 1 || jobs.doneIDs[1] != 2 {
		t.Errorf("expected both jobs marked done in order, got %v", jobs.doneIDs)
	}
}

func TestTick_StopsOnContextCancel(t *testing.T) {
	filter := mustFilter(t, notificationcontract.UserFilter{})
	job1 := newJob(1, filter, 0)
	job2 := newJob(2, filter, 0)

	ctx, cancel := context.WithCancel(context.Background())

	jobs := &fakeJobRepo{
		claimFn: func(_ context.Context, _ int) ([]*dbmodels.NotificationJob, error) {
			return []*dbmodels.NotificationJob{job1, job2}, nil
		},
		markDoneFn: func(_ context.Context, _ int64) error {
			cancel() // cancel after first job done
			return nil
		},
	}
	w := newWorker(jobs, &fakeUserRepo{}, &fakeSender{}, Config{JobBatch: 4, BatchSize: 10})

	processed, err := w.tick(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !processed {
		t.Error("expected processed=true")
	}
	if len(jobs.doneIDs) != 1 || jobs.doneIDs[0] != 1 {
		t.Errorf("expected only job 1 processed before cancel, got %v", jobs.doneIDs)
	}
}

// --- processOne ---

func TestProcessOne_Happy_PaginatesAndCheckpoints(t *testing.T) {
	users := &fakeUserRepo{
		pages: map[int64][]*dbmodels.User{
			0: makeUsers(1, 2),
			2: makeUsers(3, 4),
			4: nil,
		},
	}
	sender := &fakeSender{}
	jobs := &fakeJobRepo{}

	filter := mustFilter(t, notificationcontract.UserFilter{
		Roles:          []string{"manager"},
		ExcludeBlocked: true,
		OnlyTelegram:   true,
		UserIDs:        []int64{10, 20},
	})
	job := newJob(42, filter, 0)

	w := newWorker(jobs, users, sender, Config{BatchSize: 2})
	w.processOne(context.Background(), job)

	// SendBulk called twice with correct payload
	if len(sender.calls) != 2 {
		t.Fatalf("SendBulk calls: want 2, got %d", len(sender.calls))
	}
	if sender.calls[0].notifType != "season_available" || sender.calls[0].recipients[0].Text != "hello" {
		t.Errorf("first call payload mismatch: %+v", sender.calls[0])
	}
	if len(sender.calls[0].recipients) != 2 || sender.calls[0].recipients[0].UserID != 1 {
		t.Errorf("first call recipients mismatch: %+v", sender.calls[0].recipients)
	}

	// Cursor checkpointed after each non-empty page
	if len(jobs.updateCursorCalls) != 2 ||
		jobs.updateCursorCalls[0] != (cursorCall{42, 2}) ||
		jobs.updateCursorCalls[1] != (cursorCall{42, 4}) {
		t.Errorf("cursor checkpoints mismatch: %+v", jobs.updateCursorCalls)
	}

	// Filter mapping
	want := repository.UserFilter{
		IDs:              []int64{10, 20},
		Roles:            []string{"manager"},
		ExcludeBlocked:   true,
		OnlyWithTelegram: true,
	}
	if !equalUserFilter(users.lastFilter, want) {
		t.Errorf("user filter: want %+v, got %+v", want, users.lastFilter)
	}

	// MarkDone, no failures
	if len(jobs.doneIDs) != 1 || jobs.doneIDs[0] != 42 {
		t.Errorf("expected MarkDone(42), got %v", jobs.doneIDs)
	}
	if len(jobs.failedCalls) != 0 {
		t.Errorf("expected no failures, got %+v", jobs.failedCalls)
	}
}

func TestProcessOne_ResumesFromCursor(t *testing.T) {
	users := &fakeUserRepo{
		pages: map[int64][]*dbmodels.User{
			5: makeUsers(6, 7),
			7: nil,
		},
	}
	sender := &fakeSender{}
	jobs := &fakeJobRepo{}

	filter := mustFilter(t, notificationcontract.UserFilter{})
	job := newJob(1, filter, 0)
	job.Cursor = 5

	w := newWorker(jobs, users, sender, Config{BatchSize: 10})
	w.processOne(context.Background(), job)

	if len(users.listCalls) == 0 || users.listCalls[0].afterID != 5 {
		t.Errorf("expected first ListByFilter afterID=5, got %+v", users.listCalls)
	}
	if len(jobs.doneIDs) != 1 {
		t.Errorf("expected MarkDone, got %v", jobs.doneIDs)
	}
}

func TestProcessOne_EmptyAudience_MarksDoneWithoutSending(t *testing.T) {
	jobs := &fakeJobRepo{}
	users := &fakeUserRepo{} // returns nil page
	sender := &fakeSender{}

	job := newJob(1, mustFilter(t, notificationcontract.UserFilter{}), 0)

	w := newWorker(jobs, users, sender, Config{BatchSize: 10})
	w.processOne(context.Background(), job)

	if len(sender.calls) != 0 {
		t.Errorf("expected no SendBulk, got %d", len(sender.calls))
	}
	if len(jobs.updateCursorCalls) != 0 {
		t.Errorf("expected no cursor updates, got %+v", jobs.updateCursorCalls)
	}
	if len(jobs.doneIDs) != 1 {
		t.Errorf("expected MarkDone, got %v", jobs.doneIDs)
	}
}

func TestProcessOne_BadFilterJSON_TerminalFailure(t *testing.T) {
	jobs := &fakeJobRepo{}
	users := &fakeUserRepo{}
	sender := &fakeSender{}

	job := newJob(7, json.RawMessage(`not json`), 1)

	w := newWorker(jobs, users, sender, Config{BatchSize: 10, MaxAttempts: 5})
	w.processOne(context.Background(), job)

	if len(jobs.failedCalls) != 1 {
		t.Fatalf("expected 1 MarkFailed, got %+v", jobs.failedCalls)
	}
	got := jobs.failedCalls[0]
	if got.id != 7 || got.attempts != 2 || !got.terminal {
		t.Errorf("expected terminal failure id=7 attempts=2, got %+v", got)
	}
	if len(jobs.doneIDs) != 0 || len(sender.calls) != 0 {
		t.Errorf("expected no work past decode error")
	}
}

func TestProcessOne_ListError_RetryableFailure(t *testing.T) {
	listErr := errors.New("list down")
	users := &fakeUserRepo{
		listFn: func(_ context.Context, _ repository.UserFilter, _ int64, _ int) ([]*dbmodels.User, error) {
			return nil, listErr
		},
	}
	jobs := &fakeJobRepo{}
	sender := &fakeSender{}

	job := newJob(1, mustFilter(t, notificationcontract.UserFilter{}), 0)

	w := newWorker(jobs, users, sender, Config{BatchSize: 10, MaxAttempts: 5})
	w.processOne(context.Background(), job)

	if len(jobs.failedCalls) != 1 {
		t.Fatalf("expected 1 MarkFailed, got %+v", jobs.failedCalls)
	}
	got := jobs.failedCalls[0]
	if got.id != 1 || got.attempts != 1 || got.terminal {
		t.Errorf("want non-terminal attempt 1, got %+v", got)
	}
	if got.errMsg == "" || !contains(got.errMsg, "list down") {
		t.Errorf("expected error message to wrap underlying cause, got %q", got.errMsg)
	}
}

func TestProcessOne_SendError_TerminalAtMaxAttempts(t *testing.T) {
	users := &fakeUserRepo{pages: map[int64][]*dbmodels.User{0: makeUsers(1)}}
	sender := &fakeSender{
		sendBulk: func(_ context.Context, _ string, _ []notifier.BulkRecipient) error {
			return errors.New("notifier down")
		},
	}
	jobs := &fakeJobRepo{}

	job := newJob(9, mustFilter(t, notificationcontract.UserFilter{}), 4) // 4 prior + 1 = 5 = MaxAttempts

	w := newWorker(jobs, users, sender, Config{BatchSize: 10, MaxAttempts: 5})
	w.processOne(context.Background(), job)

	if len(jobs.failedCalls) != 1 {
		t.Fatalf("expected 1 MarkFailed, got %+v", jobs.failedCalls)
	}
	got := jobs.failedCalls[0]
	if got.attempts != 5 || !got.terminal {
		t.Errorf("expected terminal attempts=5, got %+v", got)
	}
	if len(jobs.updateCursorCalls) != 0 {
		t.Errorf("send failed before cursor update, got %+v", jobs.updateCursorCalls)
	}
}

func TestProcessOne_CursorUpdateError_BailsWithoutDoneOrFailed(t *testing.T) {
	users := &fakeUserRepo{pages: map[int64][]*dbmodels.User{0: makeUsers(1)}}
	jobs := &fakeJobRepo{
		updateCursorFn: func(_ context.Context, _, _ int64) error {
			return errors.New("checkpoint down")
		},
	}
	sender := &fakeSender{}

	job := newJob(3, mustFilter(t, notificationcontract.UserFilter{}), 0)

	w := newWorker(jobs, users, sender, Config{BatchSize: 10, MaxAttempts: 5})
	w.processOne(context.Background(), job)

	if len(sender.calls) != 1 {
		t.Errorf("expected 1 SendBulk, got %d", len(sender.calls))
	}
	if len(jobs.doneIDs) != 0 {
		t.Errorf("expected no MarkDone after cursor failure, got %v", jobs.doneIDs)
	}
	if len(jobs.failedCalls) != 0 {
		t.Errorf("expected no MarkFailed (lease timeout reclaims), got %+v", jobs.failedCalls)
	}
}

func TestProcessOne_ContextCancelledBetweenBatches_NoMarkDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	users := &fakeUserRepo{
		pages: map[int64][]*dbmodels.User{
			0: makeUsers(1, 2),
			2: makeUsers(3, 4),
		},
	}
	jobs := &fakeJobRepo{
		updateCursorFn: func(_ context.Context, _, _ int64) error {
			cancel() // cancel right after first checkpoint
			return nil
		},
	}
	sender := &fakeSender{}

	job := newJob(1, mustFilter(t, notificationcontract.UserFilter{}), 0)

	w := newWorker(jobs, users, sender, Config{BatchSize: 2, MaxAttempts: 5})
	w.processOne(ctx, job)

	if len(sender.calls) != 1 {
		t.Errorf("expected exactly 1 batch sent before cancel, got %d", len(sender.calls))
	}
	if len(jobs.doneIDs) != 0 {
		t.Errorf("expected no MarkDone after cancel mid-fanout, got %v", jobs.doneIDs)
	}
	if len(jobs.failedCalls) != 0 {
		t.Errorf("cancellation is not a failure, got %+v", jobs.failedCalls)
	}
}

// --- helpers ---

func equalUserFilter(a, b repository.UserFilter) bool {
	if a.ExcludeBlocked != b.ExcludeBlocked || a.OnlyWithTelegram != b.OnlyWithTelegram {
		return false
	}
	if !equalInt64Slice(a.IDs, b.IDs) || !equalStringSlice(a.Roles, b.Roles) {
		return false
	}
	return true
}

func equalInt64Slice(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
