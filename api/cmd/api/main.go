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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mod/internal/api"
	"go.mod/internal/config"
	"go.mod/internal/db"
	"go.mod/internal/db/repository"
	apimiddleware "go.mod/internal/middleware"
	"go.mod/internal/notifier"
	"go.mod/internal/services"
	authservice "go.mod/internal/services/auth"
	balanceservice "go.mod/internal/services/balance"
	notificationservice "go.mod/internal/services/notification"
	rewardservice "go.mod/internal/services/reward"
	seasonservice "go.mod/internal/services/season"
	sessionservice "go.mod/internal/services/session"
	submissionservice "go.mod/internal/services/submission"
	taskservice "go.mod/internal/services/task"
	templateservice "go.mod/internal/services/template"
	userservice "go.mod/internal/services/user"
	notificationworker "go.mod/internal/worker/notification"
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
	notifJobRepo := repository.NewNotificationJobRepository(bunDB, logger, time.Duration(cfg.NotificationClaimStaleSeconds)*time.Second)

	userSvc := userservice.NewService(userRepo, logger)
	sessionSvc := sessionservice.NewService(sessionRepo, logger)
	authSvc := authservice.NewService(userSvc, sessionSvc, logger, cfg)

	var notifySender notifier.Sender = notifier.Noop{}
	if cfg.NotifierAddr != "" {
		nc, conn, err := notifier.NewClient(cfg.NotifierAddr)
		if err != nil {
			logger.Error("failed to connect to notifier", slog.Any("error", err))
			os.Exit(1)
		}
		defer conn.Close()
		notifySender = nc
		logger.Info("notifier connected", slog.String("addr", cfg.NotifierAddr))
	}

	templateRepo := repository.NewTemplateRepository(bunDB, logger)
	templateSvc := templateservice.NewService(templateRepo, logger)
	notifSvc := notificationservice.NewService(notifJobRepo, bunDB, templateSvc, logger)
	seasonSvc := seasonservice.NewService(seasonRepo, bunDB, notifSvc, logger)
	balanceSvc := balanceservice.NewService(balanceRepo, logger)
	taskSvc := taskservice.NewService(taskRepo, logger)
	submissionSvc := submissionservice.NewService(submissionRepo, taskSvc, balanceSvc, notifSvc, logger)
	rewardSvc := rewardservice.NewService(rewardRepo, balanceSvc, notifSvc, logger)

	svc := &services.Container{
		Auth:       authSvc,
		User:       userSvc,
		Session:    sessionSvc,
		Season:     seasonSvc,
		Balance:    balanceSvc,
		Task:       taskSvc,
		Submission: submissionSvc,
		Reward:     rewardSvc,
		Template:   templateSvc,
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

	// Worker context: separate from request context so HTTP shutdown and worker shutdown
	// can be coordinated independently. Cancelled when the process receives SIGINT/SIGTERM.
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	worker := notificationworker.New(notifJobRepo, userRepo, notifySender, notificationworker.Config{
		JobBatch: cfg.NotificationJobBatch,
	}, logger)
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		worker.Run(workerCtx)
	}()

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsSrv := &http.Server{
		Addr:        cfg.MetricsURL,
		Handler:     metricsMux,
		ReadTimeout: 5 * time.Second,
		IdleTimeout: 60 * time.Second,
	}
	srvDone := make(chan struct{})
	go func() {
		defer close(srvDone)
		logger.Info("server starting", slog.String("addr", cfg.ApiURL), slog.String("env", *env))
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", slog.Any("error", err))
			quit <- syscall.SIGTERM
		}
	}()

	metricsDone := make(chan struct{})
	go func() {
		defer close(metricsDone)
		logger.Info("metrics server starting", slog.String("addr", cfg.MetricsURL))
		if err := metricsSrv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			logger.Error("metrics server error", slog.Any("error", err))
			quit <- syscall.SIGTERM
		}
	}()

	<-quit
	logger.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Завершаем оба сервера параллельно
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("graceful shutdown failed", slog.Any("error", err))
		}
		if err := metricsSrv.Shutdown(ctx); err != nil {
			logger.Error("metrics server shutdown failed", slog.Any("error", err))
		}
	}()

	<-srvDone
	<-metricsDone
	<-shutdownDone

	// Stop the worker after the HTTP server: any in-flight request that enqueued a job
	// has already committed, so the worker can keep draining until ctx expires.
	workerCancel()
	select {
	case <-workerDone:
	case <-ctx.Done():
		logger.Warn("notification worker did not stop within shutdown deadline")
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
