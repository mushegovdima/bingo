package services

import (
	authservice "go.mod/internal/services/auth"
	balanceservice "go.mod/internal/services/balance"
	seasonservice "go.mod/internal/services/season"
	rewardservice "go.mod/internal/services/reward"
	sessionservice "go.mod/internal/services/session"
	submissionservice "go.mod/internal/services/submission"
	taskservice "go.mod/internal/services/task"
	userservice "go.mod/internal/services/user"
)

type Container struct {
	Auth       *authservice.AuthService
	User       *userservice.UserService
	Session    *sessionservice.SessionService
	Season   *seasonservice.SeasonService
	Balance    *balanceservice.BalanceService
	Task       *taskservice.TaskService
	Submission *submissionservice.SubmissionService
	Reward     *rewardservice.RewardService
}
