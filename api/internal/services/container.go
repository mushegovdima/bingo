package services

import (
	authservice "go.mod/internal/services/auth"
	balanceservice "go.mod/internal/services/balance"
	rewardservice "go.mod/internal/services/reward"
	seasonservice "go.mod/internal/services/season"
	sessionservice "go.mod/internal/services/session"
	submissionservice "go.mod/internal/services/submission"
	taskservice "go.mod/internal/services/task"
	templateservice "go.mod/internal/services/template"
	userservice "go.mod/internal/services/user"
)

type Container struct {
	Auth       *authservice.AuthService
	User       *userservice.UserService
	Session    *sessionservice.SessionService
	Season     *seasonservice.SeasonService
	Balance    *balanceservice.BalanceService
	Task       *taskservice.TaskService
	Submission *submissionservice.SubmissionService
	Reward     *rewardservice.RewardService
	Template   *templateservice.Service
}
