package task_test

import (
	"errors"
	"testing"

	taskdomain "go.mod/internal/domain/task"
)

func TestTask_Validate(t *testing.T) {
	cases := []struct {
		name    string
		task    taskdomain.Task
		wantErr error
	}{
		{"ok", taskdomain.Task{Title: "Read a book", RewardCoins: 100}, nil},
		{"empty title", taskdomain.Task{RewardCoins: 100}, taskdomain.ErrEmptyTitle},
		{"negative reward", taskdomain.Task{Title: "x", RewardCoins: -1}, taskdomain.ErrNegativeReward},
		{"zero reward is allowed", taskdomain.Task{Title: "x", RewardCoins: 0}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.task.Validate()
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got err=%v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestTask_ActivateDeactivate(t *testing.T) {
	tk := &taskdomain.Task{}
	tk.Activate()
	if !tk.IsActive {
		t.Errorf("Activate did not set IsActive")
	}
	tk.Deactivate()
	if tk.IsActive {
		t.Errorf("Deactivate did not clear IsActive")
	}
}
