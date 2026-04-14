package main

import (
	"context"
	"encoding/gob"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mod/internal/api"
	"go.mod/internal/config"
	"go.mod/internal/db"
	"go.mod/internal/db/repository"
	apimiddleware "go.mod/internal/middleware"
	"go.mod/internal/services"
	authservice "go.mod/internal/services/auth"
	balanceservice "go.mod/internal/services/balance"
	seasonservice "go.mod/internal/services/season"
	rewardservice "go.mod/internal/services/reward"
	sessionservice "go.mod/internal/services/session"
	submissionservice "go.mod/internal/services/submission"
	taskservice "go.mod/internal/services/task"
	userservice "go.mod/internal/services/user"
)

const (
	envDev = "dev"
)

func init() {
	// gorilla/sessions uses gob encoding for session values;
	// register all types stored in session.Values.
	gob.Register(int64(0))
}

func main() {
	env := flag.String("env", "prod", "environment (dev|prod)")
	configPath := flag.String("config", "", "path to config file (default: {env}.env)")
	flag.Parse()

	cfg, err := config.LoadConfig(*env, *configPath)
	if err != nil {
		slog.Default().Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	logger := setupLogger(*env)

	database, err := db.NewDB(cfg, logger)
	if err != nil {
		logger.Error("failed to connect to database", slog.Any("error", err))
		os.Exit(1)
	}

	defer database.Close()
	logger.Info("database connected")

	bunDB := database.DB()

	userRepo := repository.NewUserRepository(context.Background(), bunDB, logger)
	sessionRepo := repository.NewSessionRepository(bunDB, logger)
	seasonRepo := repository.NewSeasonRepository(bunDB, logger)
	balanceRepo := repository.NewBalanceRepository(bunDB, logger)
	taskRepo := repository.NewTaskRepository(bunDB, logger)
	submissionRepo := repository.NewTaskSubmissionRepository(bunDB, logger)
	rewardRepo := repository.NewRewardRepository(bunDB, logger)

	userSvc := userservice.NewService(userRepo, logger, cfg)
	sessionSvc := sessionservice.NewService(sessionRepo, logger)
	authSvc := authservice.NewService(userSvc, sessionSvc, logger, cfg)
	seasonSvc := seasonservice.NewService(seasonRepo, logger)
	balanceSvc := balanceservice.NewService(balanceRepo, logger)
	taskSvc := taskservice.NewService(taskRepo, logger)
	submissionSvc := submissionservice.NewService(submissionRepo, taskSvc, balanceSvc, logger)
	rewardSvc := rewardservice.NewService(rewardRepo, balanceSvc, logger)

	svc := &services.Container{
		Auth:       authSvc,
		User:       userSvc,
		Session:    sessionSvc,
		Season:   seasonSvc,
		Balance:    balanceSvc,
		Task:       taskSvc,
		Submission: submissionSvc,
		Reward:     rewardSvc,
	}

	store := apimiddleware.NewStore(*env, *cfg)
	router := api.NewRouter(svc, cfg, store, logger)

	chi.Walk(router, func(method, route string, handler http.Handler, _ ...func(http.Handler) http.Handler) error {
		logger.Debug("Route", slog.String(method, route))
		return nil
	})

	srv := &http.Server{
		Addr:         cfg.ApiURL,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("server starting", slog.String("addr", cfg.ApiURL), slog.String("env", *env))
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-quit
	logger.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", slog.Any("error", err))
	}

	logger.Info("server stopped")
}

func setupLogger(env string) *slog.Logger {
	switch env {
	case envDev:
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	default:
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	}
}
