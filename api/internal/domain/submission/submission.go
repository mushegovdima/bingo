package submission

import (
	"errors"
	"time"
)

// Domain invariant errors.
var (
	// ErrNotFound means a submission lookup found no record.
	ErrNotFound = errors.New("submission not found")
	// ErrNotPending means an action was requested on a submission that has
	// already been reviewed.
	ErrNotPending = errors.New("submission is not pending")
	// ErrEmptyComment means a rejection was issued without a comment.
	ErrEmptyComment = errors.New("rejection requires a comment")
	// ErrAlreadySubmitted means the resident already has an active submission for this task.
	ErrAlreadySubmitted = errors.New("task already submitted")
)

// TaskSubmissionStatus enumerates review states for a TaskSubmission.
type TaskSubmissionStatus string

const (
	SubmissionPending  TaskSubmissionStatus = "pending"
	SubmissionApproved TaskSubmissionStatus = "approved"
	SubmissionRejected TaskSubmissionStatus = "rejected"
)

// TaskSubmission is a Resident's report against a Task, awaiting (or having
// completed) Manager review. State transitions: pending → approved | rejected.
type TaskSubmission struct {
	ID            int64                `json:"id"`
	UserID        int64                `json:"user_id"`
	TaskID        int64                `json:"task_id"`
	SeasonID      int64                `json:"season_id"`
	Status        TaskSubmissionStatus `json:"status"`
	Comment       string               `json:"comment"`
	ReviewComment string               `json:"review_comment"`
	ReviewerID    *int64               `json:"reviewer_id"`
	SubmittedAt   time.Time            `json:"submitted_at"`
	ReviewedAt    *time.Time           `json:"reviewed_at"`
}

// NewPending constructs a TaskSubmission in the initial pending state.
func NewPending(userID, taskID int64, comment string, now time.Time) *TaskSubmission {
	return &TaskSubmission{
		UserID:      userID,
		TaskID:      taskID,
		Status:      SubmissionPending,
		Comment:     comment,
		SubmittedAt: now,
	}
}

// NewApproved constructs a pre-approved TaskSubmission (manager flow).
// SubmittedAt and ReviewedAt are both set to now.
func NewApproved(userID, taskID, reviewerID int64, now time.Time) *TaskSubmission {
	rid := reviewerID
	t := now
	return &TaskSubmission{
		UserID:      userID,
		TaskID:      taskID,
		Status:      SubmissionApproved,
		ReviewerID:  &rid,
		SubmittedAt: now,
		ReviewedAt:  &t,
	}
}

// IsPending reports whether the submission still awaits review.
func (s *TaskSubmission) IsPending() bool { return s.Status == SubmissionPending }

// IsTerminal reports whether the submission has been reviewed.
func (s *TaskSubmission) IsTerminal() bool {
	return s.Status == SubmissionApproved || s.Status == SubmissionRejected
}

// Approve transitions a pending submission to approved.
// Returns ErrNotPending if the submission has already been reviewed.
func (s *TaskSubmission) Approve(reviewerID int64, now time.Time) error {
	if !s.IsPending() {
		return ErrNotPending
	}
	rid := reviewerID
	t := now
	s.Status = SubmissionApproved
	s.ReviewerID = &rid
	s.ReviewedAt = &t
	s.ReviewComment = ""
	return nil
}

// Reject transitions a pending submission to rejected. A non-empty comment
// is required so the resident understands why.
func (s *TaskSubmission) Reject(reviewerID int64, comment string, now time.Time) error {
	if !s.IsPending() {
		return ErrNotPending
	}
	if comment == "" {
		return ErrEmptyComment
	}
	rid := reviewerID
	t := now
	s.Status = SubmissionRejected
	s.ReviewerID = &rid
	s.ReviewedAt = &t
	s.ReviewComment = comment
	return nil
}
