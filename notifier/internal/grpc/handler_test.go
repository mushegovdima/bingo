package grpchandler

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"notifier/internal/db/repository"
	pb "notifier/internal/pb/notification/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeRepo is an inline test double for notificationRepo.
type fakeRepo struct {
	inserted     *repository.Notification
	bulkInserted []repository.Notification
	err          error
}

func (f *fakeRepo) Insert(_ context.Context, e *repository.Notification) error {
	if f.err != nil {
		return f.err
	}
	f.inserted = e
	return nil
}

func (f *fakeRepo) BulkInsert(_ context.Context, entries []repository.Notification) error {
	if f.err != nil {
		return f.err
	}
	f.bulkInserted = append(f.bulkInserted, entries...)
	return nil
}

const (
	validID         = "550e8400-e29b-41d4-a716-446655440000"
	validTelegramID = int64(123456789)
	validUserID     = int64(42)
	validText       = "hello"
	validType       = "welcome_message"
)

func newHandler(repo notificationRepo) *Handler {
	return New(repo, slog.Default())
}

func validRequest() *pb.SendNotificationRequest {
	return &pb.SendNotificationRequest{
		Id:         validID,
		Type:       validType,
		UserId:     validUserID,
		TelegramId: validTelegramID,
		Text:       validText,
	}
}

// --- Send tests ---

func TestSend_SavesEntryToOutbox(t *testing.T) {
	repo := &fakeRepo{}
	h := newHandler(repo)

	resp, err := h.Send(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp.Id != validID {
		t.Errorf("response id = %q, want %q", resp.Id, validID)
	}

	e := repo.inserted
	if e == nil {
		t.Fatal("expected entry to be inserted, got nil")
	}
	if e.ID != validID || e.UserID != validUserID || e.TelegramID != validTelegramID ||
		e.Text != validText || e.Type != validType {
		t.Errorf("entry mismatch: %+v", e)
	}
}

func TestSend_SendAt_IsNow(t *testing.T) {
	repo := &fakeRepo{}
	h := newHandler(repo)

	before := time.Now()
	_, err := h.Send(context.Background(), validRequest())
	after := time.Now()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sendAt := repo.inserted.SendAt
	if sendAt.Before(before) || sendAt.After(after) {
		t.Errorf("send_at %v not in expected range [%v, %v]", sendAt, before, after)
	}
}

func TestSend_RepoError_ReturnsInternalStatus(t *testing.T) {
	repo := &fakeRepo{err: errors.New("db unavailable")}
	h := newHandler(repo)

	_, err := h.Send(context.Background(), validRequest())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got: %T", err)
	}
	if st.Code() != codes.Internal {
		t.Errorf("status code = %v, want %v", st.Code(), codes.Internal)
	}
}

// --- SendBulk tests (unary) ---

func TestSendBulk_FansOutSharedBodyToAllRecipients(t *testing.T) {
	repo := &fakeRepo{}
	h := newHandler(repo)

	req := &pb.SendBulkNotificationRequest{
		Notifications: []*pb.SendNotificationRequest{
			{Id: "550e8400-e29b-41d4-a716-446655440001", Type: validType, UserId: 1, TelegramId: 111, Text: "shared body"},
			{Id: "550e8400-e29b-41d4-a716-446655440002", Type: validType, UserId: 2, TelegramId: 222, Text: "shared body"},
			{Id: "550e8400-e29b-41d4-a716-446655440003", Type: validType, UserId: 3, TelegramId: 333, Text: "shared body"},
		},
	}

	resp, err := h.SendBulk(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Ids) != len(req.Notifications) {
		t.Fatalf("response ids = %d, want %d", len(resp.Ids), len(req.Notifications))
	}
	if len(repo.bulkInserted) != len(req.Notifications) {
		t.Fatalf("bulk inserted = %d, want %d", len(repo.bulkInserted), len(req.Notifications))
	}
	for i, r := range req.Notifications {
		e := repo.bulkInserted[i]
		if e.ID != r.Id {
			t.Errorf("entry[%d].ID = %q, want %q", i, e.ID, r.Id)
		}
		if e.UserID != r.UserId {
			t.Errorf("entry[%d].UserID = %d, want %d", i, e.UserID, r.UserId)
		}
		if e.TelegramID != r.TelegramId {
			t.Errorf("entry[%d].TelegramID = %d, want %d", i, e.TelegramID, r.TelegramId)
		}
		if e.Text != r.Text {
			t.Errorf("entry[%d].Text = %q, want %q", i, e.Text, r.Text)
		}
		if e.Type != r.Type {
			t.Errorf("entry[%d].Type = %q, want %q", i, e.Type, r.Type)
		}
		if resp.Ids[i] != r.Id {
			t.Errorf("resp.Ids[%d] = %q, want %q", i, resp.Ids[i], r.Id)
		}
	}
}

func TestSendBulk_RejectsEmptyRecipients(t *testing.T) {
	h := newHandler(&fakeRepo{})

	_, err := h.SendBulk(context.Background(), &pb.SendBulkNotificationRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Errorf("status = %v, want InvalidArgument", st.Code())
	}
}

func TestSendBulk_RepoError_ReturnsInternalStatus(t *testing.T) {
	repo := &fakeRepo{err: errors.New("db unavailable")}
	h := newHandler(repo)

	_, err := h.SendBulk(context.Background(), &pb.SendBulkNotificationRequest{
		Notifications: []*pb.SendNotificationRequest{
			{Id: validID, Type: validType, UserId: 1, TelegramId: 111, Text: "body"},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.Internal {
		t.Errorf("status = %v, want Internal", st.Code())
	}
}

func TestSendBulk_SendAt_IsSharedNow(t *testing.T) {
	repo := &fakeRepo{}
	h := newHandler(repo)

	before := time.Now()
	_, err := h.SendBulk(context.Background(), &pb.SendBulkNotificationRequest{
		Notifications: []*pb.SendNotificationRequest{
			{Id: "550e8400-e29b-41d4-a716-446655440001", Type: validType, UserId: 1, TelegramId: 111, Text: "body"},
			{Id: "550e8400-e29b-41d4-a716-446655440002", Type: validType, UserId: 2, TelegramId: 222, Text: "body"},
		},
	})
	after := time.Now()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	first := repo.bulkInserted[0].SendAt
	if first.Before(before) || first.After(after) {
		t.Errorf("send_at %v not in [%v, %v]", first, before, after)
	}
	if !repo.bulkInserted[1].SendAt.Equal(first) {
		t.Errorf("expected all recipients to share the same send_at; got %v vs %v",
			repo.bulkInserted[1].SendAt, first)
	}
}
