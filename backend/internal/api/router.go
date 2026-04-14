package api

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/gorilla/sessions"
	adminapi "go.mod/internal/api/admin"
	authapi "go.mod/internal/api/auth"
	balanceapi "go.mod/internal/api/balance"
	seasonapi "go.mod/internal/api/season"
	claimapi "go.mod/internal/api/claim"
	rewardapi "go.mod/internal/api/reward"
	submissionapi "go.mod/internal/api/submission"
	taskapi "go.mod/internal/api/task"
	"go.mod/internal/config"
	"go.mod/internal/domain"
	apimiddleware "go.mod/internal/middleware"
	"go.mod/internal/services"
)

func NewRouter(svc *services.Container, cfg *config.Config, store *sessions.CookieStore, logger *slog.Logger) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.ClientURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(apimiddleware.SessionRefresh(store))

	managerOnly := apimiddleware.RequireRole(svc.User, domain.Manager)

	authHandler := authapi.NewHandler(svc.Auth, svc.User, svc.Session, store, cfg, logger)
	seasonHandler := seasonapi.NewHandler(svc.Season, logger)
	balanceHandler := balanceapi.NewHandler(svc.Balance, logger)
	taskHandler := taskapi.NewHandler(svc.Task, logger)
	submissionHandler := submissionapi.NewHandler(svc.Submission, logger)
	rewardHandler := rewardapi.NewHandler(svc.Reward, logger)
	claimHandler := claimapi.NewHandler(svc.Reward, logger)
	adminHandler := adminapi.NewHandler(svc.User, logger)

	// Auth роуты: /login, /logout — открытые; /me — только для залогиненных; /impersonate — только менеджеры
	r.Mount("/auth", authHandler.Routes(apimiddleware.RequireAuth, managerOnly))

	// Закрытые роуты — только для залогиненных
	r.Group(func(r chi.Router) {
		r.Use(apimiddleware.RequireAuth)
		r.Mount("/seasons", seasonHandler.Routes())
		r.Mount("/balance", balanceHandler.Routes(managerOnly))
		r.Mount("/tasks", taskHandler.Routes(managerOnly))
		r.Mount("/submissions", submissionHandler.Routes(managerOnly))
		r.Mount("/rewards", rewardHandler.Routes(managerOnly))
		r.Mount("/claims", claimHandler.Routes(managerOnly))
	})

	// Admin роуты — только для менеджеров
	r.Group(func(r chi.Router) {
		r.Use(apimiddleware.RequireAuth)
		r.Use(managerOnly)
		r.Mount("/admin", adminHandler.Routes())
	})

	return r
}
