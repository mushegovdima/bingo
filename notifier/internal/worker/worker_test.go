package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"notifier/internal/db/repository"
	"notifier/internal/sender"
)

// ---- fakes ----

type fakeRepo struct {
	fetchAndReserve func(ctx context.Context, limit int, ttl time.Duration) ([]repository.Notification, error)
	markProcessed   func(ctx context.Context, id string) error
}

func (r *fakeRepo) FetchAndReserve(ctx context.Context, limit int, ttl time.Duration) ([]repository.Notification, error) {
	return r.fetchAndReserve(ctx, limit, ttl)
}

func (r *fakeRepo) MarkProcessed(ctx context.Context, id string) error {
	return r.markProcessed(ctx, id)
}

type fakeSender struct {
	send func(ctx context.Context, chatID int64, text string) error
}

func (s *fakeSender) Send(ctx context.Context, chatID int64, text string) error {
	return s.send(ctx, chatID, text)
}

var discardLogger = slog.New(slog.NewTextHandler(nopWriter{}, nil))

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func newWorker(repo notificationRepo, sender messageSender, batchSize int) *Worker {
	return New(repo, sender, time.Hour, Config{BatchSize: batchSize, ReservationTTL: 5 * time.Second}, discardLogger)
}

// ---- tick tests ----

func TestTick_ReturnsFalseWhenBatchIsEmpty(t *testing.T) {
	repo := &fakeRepo{
		fetchAndReserve: func(_ context.Context, _ int, _ time.Duration) ([]repository.Notification, error) {
			return nil, nil
		},
		markProcessed: func(_ context.Context, _ string) error {
			t.Error("markProcessed must not be called on empty batch")
			return nil
		},
	}
	sender := &fakeSender{
		send: func(_ context.Context, _ int64, _ string) error {
			t.Error("Send must not be called on empty batch")
			return nil
		},
	}

	w := newWorker(repo, sender, 10)
	if w.tick(context.Background()) {
		t.Error("expected tick to return false on empty batch")
	}
}

func TestTick_ReturnsFalseOnFetchError(t *testing.T) {
	fetchErr := errors.New("db down")
	repo := &fakeRepo{
		fetchAndReserve: func(_ context.Context, _ int, _ time.Duration) ([]repository.Notification, error) {
			return nil, fetchErr
		},
		markProcessed: func(_ context.Context, _ string) error {
			t.Error("markProcessed must not be called on fetch error")
			return nil
		},
	}
	sender := &fakeSender{
		send: func(_ context.Context, _ int64, _ string) error {
			t.Error("Send must not be called on fetch error")
			return nil
		},
	}

	w := newWorker(repo, sender, 10)
	if w.tick(context.Background()) {
		t.Error("expected tick to return false on fetch error")
	}
}

func TestTick_SendsEachEntryAndMarksProcessed(t *testing.T) {
	entries := []repository.Notification{
		{ID: "aaa", TelegramID: 1, Text: "hello"},
		{ID: "bbb", TelegramID: 2, Text: "world"},
	}

	sentTo := map[int64]string{}
	markedProcessed := map[string]bool{}

	repo := &fakeRepo{
		fetchAndReserve: func(_ context.Context, _ int, _ time.Duration) ([]repository.Notification, error) {
			return entries, nil
		},
		markProcessed: func(_ context.Context, id string) error {
			markedProcessed[id] = true
			return nil
		},
	}
	sender := &fakeSender{
		send: func(_ context.Context, chatID int64, text string) error {
			sentTo[chatID] = text
			return nil
		},
	}

	w := newWorker(repo, sender, 10)
	w.tick(context.Background())

	for _, e := range entries {
		if sentTo[e.TelegramID] != e.Text {
			t.Errorf("entry %s: expected Send(%d, %q), not called or wrong text", e.ID, e.TelegramID, e.Text)
		}
		if !markedProcessed[e.ID] {
			t.Errorf("entry %s: expected MarkProcessed to be called", e.ID)
		}
	}
}

func TestTick_SkipsMarkProcessedWhenSendFails(t *testing.T) {
	entries := []repository.Notification{
		{ID: "aaa", TelegramID: 1, Text: "hello"},
	}

	repo := &fakeRepo{
		fetchAndReserve: func(_ context.Context, _ int, _ time.Duration) ([]repository.Notification, error) {
			return entries, nil
		},
		markProcessed: func(_ context.Context, _ string) error {
			t.Error("MarkProcessed must not be called when Send fails")
			return nil
		},
	}
	sender := &fakeSender{
		send: func(_ context.Context, _ int64, _ string) error {
			return errors.New("telegram unreachable")
		},
	}

	w := newWorker(repo, sender, 10)
	w.tick(context.Background()) // must not panic or call MarkProcessed
}

func TestTick_MarksProcessedOnPermanentSendError(t *testing.T) {
	entries := []repository.Notification{
		{ID: "aaa", TelegramID: 1, Text: "hello"},
		{ID: "bbb", TelegramID: 2, Text: "world"},
	}

	markedProcessed := map[string]bool{}
	repo := &fakeRepo{
		fetchAndReserve: func(_ context.Context, _ int, _ time.Duration) ([]repository.Notification, error) {
			return entries, nil
		},
		markProcessed: func(_ context.Context, id string) error {
			markedProcessed[id] = true
			return nil
		},
	}
	// Both sends fail with ErrPermanent (bot blocked, chat not found, etc.)
	fakeSnd := &fakeSender{
		send: func(_ context.Context, _ int64, _ string) error {
			return fmt.Errorf("wrap: %w", sender.ErrPermanent)
		},
	}

	w := newWorker(repo, fakeSnd, 10)
	w.tick(context.Background())

	for _, e := range entries {
		if !markedProcessed[e.ID] {
			t.Errorf("entry %s: expected MarkProcessed to be called on permanent error", e.ID)
		}
	}
}

func TestTick_DoesNotMarkProcessedOnTransientSendError(t *testing.T) {
	entries := []repository.Notification{
		{ID: "aaa", TelegramID: 1, Text: "hello"},
	}

	repo := &fakeRepo{
		fetchAndReserve: func(_ context.Context, _ int, _ time.Duration) ([]repository.Notification, error) {
			return entries, nil
		},
		markProcessed: func(_ context.Context, _ string) error {
			t.Error("MarkProcessed must not be called on transient send error")
			return nil
		},
	}
	fakeSnd := &fakeSender{
		send: func(_ context.Context, _ int64, _ string) error {
			return errors.New("network timeout") // transient, not ErrPermanent
		},
	}

	w := newWorker(repo, fakeSnd, 10)
	w.tick(context.Background())
}

func TestTick_ContinuesAfterMarkProcessedError(t *testing.T) {
	entries := []repository.Notification{
		{ID: "aaa", TelegramID: 1, Text: "first"},
		{ID: "bbb", TelegramID: 2, Text: "second"},
	}

	sentCount := 0
	repo := &fakeRepo{
		fetchAndReserve: func(_ context.Context, _ int, _ time.Duration) ([]repository.Notification, error) {
			return entries, nil
		},
		markProcessed: func(_ context.Context, _ string) error {
			return errors.New("db error")
		},
	}
	sender := &fakeSender{
		send: func(_ context.Context, _ int64, _ string) error {
			sentCount++
			return nil
		},
	}

	w := newWorker(repo, sender, 10)
	w.tick(context.Background())

	if sentCount != 2 {
		t.Errorf("expected 2 sends, got %d", sentCount)
	}
}

func TestTick_ReturnsTrueOnFullBatch(t *testing.T) {
	const batchSize = 3
	entries := make([]repository.Notification, batchSize)
	for i := range entries {
		entries[i] = repository.Notification{ID: "x", TelegramID: 1, Text: "t"}
	}

	repo := &fakeRepo{
		fetchAndReserve: func(_ context.Context, _ int, _ time.Duration) ([]repository.Notification, error) {
			return entries, nil
		},
		markProcessed: func(_ context.Context, _ string) error { return nil },
	}
	sender := &fakeSender{
		send: func(_ context.Context, _ int64, _ string) error { return nil },
	}

	w := newWorker(repo, sender, batchSize)
	if !w.tick(context.Background()) {
		t.Error("expected tick to return true when full batch received")
	}
}

func TestTick_ReturnsFalseOnPartialBatch(t *testing.T) {
	const batchSize = 10
	entries := []repository.Notification{
		{ID: "aaa", TelegramID: 1, Text: "hello"},
	}

	repo := &fakeRepo{
		fetchAndReserve: func(_ context.Context, _ int, _ time.Duration) ([]repository.Notification, error) {
			return entries, nil
		},
		markProcessed: func(_ context.Context, _ string) error { return nil },
	}
	sender := &fakeSender{
		send: func(_ context.Context, _ int64, _ string) error { return nil },
	}

	w := newWorker(repo, sender, batchSize)
	if w.tick(context.Background()) {
		t.Error("expected tick to return false on partial batch")
	}
}

// ---- Run tests ----

func TestRun_StopsWhenContextCancelled(t *testing.T) {
	repo := &fakeRepo{
		fetchAndReserve: func(_ context.Context, _ int, _ time.Duration) ([]repository.Notification, error) {
			return nil, nil
		},
		markProcessed: func(_ context.Context, _ string) error { return nil },
	}
	sender := &fakeSender{
		send: func(_ context.Context, _ int64, _ string) error { return nil },
	}

	w := newWorker(repo, sender, 10)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("Run did not stop within 1s after context cancellation")
	}
}

// ---- Concurrency tests ----

// TestRun_MultipleWorkersNoRaceConditions verifies that running several workers
// concurrently against the same (mutex-protected) repo produces no data races.
// Run with -race to exercise the detector.
func TestRun_MultipleWorkersNoRaceConditions(t *testing.T) {
	var callsMu sync.Mutex
	calls := 0

	repo := &fakeRepo{
		fetchAndReserve: func(_ context.Context, _ int, _ time.Duration) ([]repository.Notification, error) {
			callsMu.Lock()
			calls++
			callsMu.Unlock()
			return nil, nil
		},
		markProcessed: func(_ context.Context, _ string) error { return nil },
	}
	sender := &fakeSender{
		send: func(_ context.Context, _ int64, _ string) error { return nil },
	}

	ctx, cancel := context.WithCancel(context.Background())

	const numWorkers = 5
	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := newWorker(repo, sender, 10)
			w.Run(ctx)
		}()
	}

	time.Sleep(20 * time.Millisecond)
	cancel()
	wg.Wait()

	// sanity: each worker fetched at least once on startup
	callsMu.Lock()
	defer callsMu.Unlock()
	if calls < numWorkers {
		t.Errorf("expected at least %d FetchAndReserve calls, got %d", numWorkers, calls)
	}
}

// TestRun_MultipleWorkersProcessEachEntryOnce simulates the DB reservation pattern:
// FetchAndReserve claims entries atomically, so concurrent workers never receive
// the same entry. Verifies each entry is sent exactly once across all workers.
func TestRun_MultipleWorkersProcessEachEntryOnce(t *testing.T) {
	allEntries := []repository.Notification{
		{ID: "msg-1", TelegramID: 101, Text: "a"},
		{ID: "msg-2", TelegramID: 102, Text: "b"},
		{ID: "msg-3", TelegramID: 103, Text: "c"},
		{ID: "msg-4", TelegramID: 104, Text: "d"},
		{ID: "msg-5", TelegramID: 105, Text: "e"},
	}

	var repoMu sync.Mutex
	claimed := map[string]bool{}

	repo := &fakeRepo{
		fetchAndReserve: func(_ context.Context, limit int, _ time.Duration) ([]repository.Notification, error) {
			repoMu.Lock()
			defer repoMu.Unlock()
			var batch []repository.Notification
			for _, e := range allEntries {
				if !claimed[e.ID] {
					claimed[e.ID] = true
					batch = append(batch, e)
					if len(batch) >= limit {
						break
					}
				}
			}
			return batch, nil
		},
		markProcessed: func(_ context.Context, _ string) error { return nil },
	}

	var sentMu sync.Mutex
	sentCounts := map[int64]int{}
	sender := &fakeSender{
		send: func(_ context.Context, chatID int64, _ string) error {
			sentMu.Lock()
			sentCounts[chatID]++
			sentMu.Unlock()
			return nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := newWorker(repo, sender, 2)
			w.Run(ctx)
		}()
	}
	wg.Wait()

	sentMu.Lock()
	defer sentMu.Unlock()
	for _, e := range allEntries {
		if sentCounts[e.TelegramID] != 1 {
			t.Errorf("entry %s (chatID %d): sent %d times, expected exactly 1",
				e.ID, e.TelegramID, sentCounts[e.TelegramID])
		}
	}
}

// ---- Telegram rate-limit tests ----

// TestTick_SendBlocksLongerThanReservationTTL_EntrySkipped verifies that when
// Send blocks until the per-entry context deadline (simulating a 429 RetryAfter
// exceeding the reservation window), the entry is skipped and the next entry in
// the batch is still delivered.
func TestTick_SendBlocksLongerThanReservationTTL_EntrySkipped(t *testing.T) {
	entries := []repository.Notification{
		{ID: "slow", TelegramID: 1, Text: "blocked"},
		{ID: "fast", TelegramID: 2, Text: "ok"},
	}

	markedProcessed := map[string]bool{}

	repo := &fakeRepo{
		fetchAndReserve: func(_ context.Context, _ int, _ time.Duration) ([]repository.Notification, error) {
			return entries, nil
		},
		markProcessed: func(_ context.Context, id string) error {
			markedProcessed[id] = true
			return nil
		},
	}
	sender := &fakeSender{
		send: func(ctx context.Context, chatID int64, _ string) error {
			if chatID == 1 {
				// simulate RetryAfter > ReservationTTL: block until per-entry ctx expires
				<-ctx.Done()
				return ctx.Err()
			}
			return nil
		},
	}

	// very short TTL so the test runs fast
	w := New(repo, sender, time.Hour, Config{BatchSize: 10, ReservationTTL: 10 * time.Millisecond}, discardLogger)
	w.tick(context.Background())

	if markedProcessed["slow"] {
		t.Error("slow entry must not be marked processed when its context expired")
	}
	if !markedProcessed["fast"] {
		t.Error("fast entry must be marked processed despite the slow entry")
	}
}

// TestTick_RateLimitedEntriesDoNotBlockWorkerShutdown verifies that cancelling
// the parent context while Send is blocked (waiting out a 429 RetryAfter) causes
// the worker to exit promptly — not hang until RetryAfter elapses.
func TestTick_RateLimitedEntriesDoNotBlockWorkerShutdown(t *testing.T) {
	entries := []repository.Notification{
		{ID: "blocked", TelegramID: 1, Text: "msg"},
	}

	var fetchCalls atomic.Int32

	repo := &fakeRepo{
		fetchAndReserve: func(_ context.Context, _ int, _ time.Duration) ([]repository.Notification, error) {
			if fetchCalls.Add(1) == 1 {
				return entries, nil
			}
			return nil, nil
		},
		markProcessed: func(_ context.Context, _ string) error { return nil },
	}

	parentCtx, cancel := context.WithCancel(context.Background())

	sender := &fakeSender{
		send: func(ctx context.Context, _ int64, _ string) error {
			// simulate waiting for RetryAfter — cancel parent while blocked
			cancel()
			<-ctx.Done()
			return ctx.Err()
		},
	}

	w := New(repo, sender, time.Hour, Config{BatchSize: 10, ReservationTTL: time.Second}, discardLogger)

	done := make(chan struct{})
	go func() {
		w.Run(parentCtx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("Run did not stop within 2s after context cancellation during rate-limit wait")
	}
}

// ---- High-volume tests ----

// TestRun_DrainsManyEntriesAcrossMultipleBatches verifies that when the outbox
// holds more entries than a single batch, the worker keeps processing batches
// continuously (drain loop) until all are sent — without waiting for the next tick.
func TestRun_DrainsManyEntriesAcrossMultipleBatches(t *testing.T) {
	const total = 500
	const batchSize = 20

	allEntries := make([]repository.Notification, total)
	for i := range allEntries {
		allEntries[i] = repository.Notification{
			ID:         fmt.Sprintf("id-%d", i),
			TelegramID: int64(i + 1),
			Text:       "msg",
		}
	}

	var repoMu sync.Mutex
	cursor := 0

	repo := &fakeRepo{
		fetchAndReserve: func(_ context.Context, limit int, _ time.Duration) ([]repository.Notification, error) {
			repoMu.Lock()
			defer repoMu.Unlock()
			end := cursor + limit
			if end > len(allEntries) {
				end = len(allEntries)
			}
			batch := allEntries[cursor:end]
			cursor = end
			return batch, nil
		},
		markProcessed: func(_ context.Context, _ string) error { return nil },
	}

	var sent atomic.Int32
	sender := &fakeSender{
		send: func(_ context.Context, _ int64, _ string) error {
			sent.Add(1)
			return nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	w := New(repo, sender, time.Hour, Config{BatchSize: batchSize, ReservationTTL: 5 * time.Second}, discardLogger)

	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	// wait until all entries are sent or timeout
	deadline := time.After(2 * time.Second)
	for {
		if int(sent.Load()) >= total {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("only %d/%d entries sent within deadline", sent.Load(), total)
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	<-done

	if got := int(sent.Load()); got != total {
		t.Errorf("expected %d sends, got %d", total, got)
	}
}

// TestRun_HighVolume_MultipleWorkersNoLostEntries verifies that under high load
// with concurrent workers, every entry is delivered exactly once and nothing is lost.
func TestRun_HighVolume_MultipleWorkersNoLostEntries(t *testing.T) {
	const total = 300
	const batchSize = 15
	const numWorkers = 4

	allEntries := make([]repository.Notification, total)
	for i := range allEntries {
		allEntries[i] = repository.Notification{
			ID:         fmt.Sprintf("hv-%d", i),
			TelegramID: int64(i + 1),
			Text:       "msg",
		}
	}

	var repoMu sync.Mutex
	claimed := make([]bool, total)

	repo := &fakeRepo{
		fetchAndReserve: func(_ context.Context, limit int, _ time.Duration) ([]repository.Notification, error) {
			repoMu.Lock()
			defer repoMu.Unlock()
			var batch []repository.Notification
			for i, e := range allEntries {
				if !claimed[i] {
					claimed[i] = true
					batch = append(batch, e)
					if len(batch) >= limit {
						break
					}
				}
			}
			return batch, nil
		},
		markProcessed: func(_ context.Context, _ string) error { return nil },
	}

	var sentMu sync.Mutex
	sentIDs := make(map[string]int, total)
	sender := &fakeSender{
		send: func(_ context.Context, chatID int64, _ string) error {
			sentMu.Lock()
			sentIDs[fmt.Sprintf("hv-%d", chatID-1)]++
			sentMu.Unlock()
			return nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := New(repo, sender, time.Hour, Config{BatchSize: batchSize, ReservationTTL: 5 * time.Second}, discardLogger)
			w.Run(ctx)
		}()
	}

	// wait until all entries processed or timeout
	deadline := time.After(3 * time.Second)
	for {
		sentMu.Lock()
		n := len(sentIDs)
		sentMu.Unlock()
		if n >= total {
			break
		}
		select {
		case <-deadline:
			sentMu.Lock()
			t.Fatalf("only %d/%d entries sent within deadline", len(sentIDs), total)
			sentMu.Unlock()
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	wg.Wait()

	sentMu.Lock()
	defer sentMu.Unlock()
	for _, e := range allEntries {
		if sentIDs[e.ID] != 1 {
			t.Errorf("entry %s: sent %d times, expected exactly 1", e.ID, sentIDs[e.ID])
		}
	}
}
