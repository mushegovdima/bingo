package submission_test

import (
	"errors"
	"testing"
	"time"

	submissiondomain "go.mod/internal/domain/submission"
)

func TestNewApproved(t *testing.T) {
	now := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	sub := submissiondomain.NewApproved(1, 2, 99, now)

	if sub.Status != submissiondomain.SubmissionApproved {
		t.Errorf("Status: got %q, want approved", sub.Status)
	}
	if sub.ReviewerID == nil || *sub.ReviewerID != 99 {
		t.Errorf("ReviewerID: got %v, want 99", sub.ReviewerID)
	}
	if sub.ReviewedAt == nil || !sub.ReviewedAt.Equal(now) {
		t.Errorf("ReviewedAt: got %v, want %v", sub.ReviewedAt, now)
	}
}

func TestTaskSubmission_Approve(t *testing.T) {
	now := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)

	t.Run("transitions pending to approved", func(t *testing.T) {
		sub := submissiondomain.NewPending(1, 2, "", now)
		if err := sub.Approve(42, now.Add(time.Hour)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sub.Status != submissiondomain.SubmissionApproved {
			t.Errorf("Status: got %q", sub.Status)
		}
		if sub.ReviewerID == nil || *sub.ReviewerID != 42 {
			t.Errorf("ReviewerID: %v", sub.ReviewerID)
		}
		if sub.ReviewComment != "" {
			t.Errorf("approval should clear ReviewComment, got %q", sub.ReviewComment)
		}
	})

	t.Run("rejects non-pending submissions", func(t *testing.T) {
		for _, status := range []submissiondomain.TaskSubmissionStatus{
			submissiondomain.SubmissionApproved,
			submissiondomain.SubmissionRejected,
		} {
			sub := &submissiondomain.TaskSubmission{Status: status}
			if err := sub.Approve(1, now); !errors.Is(err, submissiondomain.ErrNotPending) {
				t.Errorf("status=%q: got err=%v, want ErrNotPending", status, err)
			}
		}
	})
}

func TestTaskSubmission_Reject(t *testing.T) {
	now := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)

	t.Run("transitions pending to rejected with comment", func(t *testing.T) {
		sub := submissiondomain.NewPending(1, 2, "", now)
		if err := sub.Reject(42, "fake photo", now); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sub.Status != submissiondomain.SubmissionRejected {
			t.Errorf("Status: got %q", sub.Status)
		}
		if sub.ReviewComment != "fake photo" {
			t.Errorf("ReviewComment: got %q", sub.ReviewComment)
		}
	})

	t.Run("requires non-empty comment", func(t *testing.T) {
		sub := submissiondomain.NewPending(1, 2, "", now)
		if err := sub.Reject(42, "", now); !errors.Is(err, submissiondomain.ErrEmptyComment) {
			t.Errorf("got err=%v, want ErrEmptyComment", err)
		}
		if sub.Status != submissiondomain.SubmissionPending {
			t.Errorf("status mutated to %q on failure", sub.Status)
		}
	})

	t.Run("rejects non-pending submissions", func(t *testing.T) {
		sub := &submissiondomain.TaskSubmission{Status: submissiondomain.SubmissionApproved}
		if err := sub.Reject(1, "x", now); !errors.Is(err, submissiondomain.ErrNotPending) {
			t.Errorf("got err=%v, want ErrNotPending", err)
		}
	})
}
