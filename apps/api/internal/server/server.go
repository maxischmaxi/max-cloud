package server

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/max-cloud/api/internal/auth"
	"github.com/max-cloud/api/internal/email"
	"github.com/max-cloud/api/internal/handler"
	"github.com/max-cloud/api/internal/orchestrator"
	"github.com/max-cloud/api/internal/store"
)

// Server holds dependencies for the API server.
type Server struct {
	logger              *slog.Logger
	store               store.ServiceStore
	authStore           store.AuthStore
	orchestrator        orchestrator.Orchestrator
	emailSender         email.Sender
	inviteExpiry        time.Duration
	devMode             bool
	devOrgUID           string
	registryURL         string
	registryJWTSecret   string
	registryTokenExpiry time.Duration
}

// New creates a new Server.
func New(logger *slog.Logger, st store.ServiceStore, authSt store.AuthStore, orch orchestrator.Orchestrator, emailSender email.Sender, inviteExpiry time.Duration, devMode bool, devOrgUID string, registryURL string, registryJWTSecret string, registryTokenExpiry time.Duration) *Server {
	return &Server{
		logger:              logger,
		store:               st,
		authStore:           authSt,
		orchestrator:        orch,
		emailSender:         emailSender,
		inviteExpiry:        inviteExpiry,
		devMode:             devMode,
		devOrgUID:           devOrgUID,
		registryURL:         registryURL,
		registryJWTSecret:   registryJWTSecret,
		registryTokenExpiry: registryTokenExpiry,
	}
}

// Router returns the configured HTTP router.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	h := handler.New(s.logger, s.store, s.authStore, s.orchestrator, s.emailSender, s.inviteExpiry, s.devMode, s.registryURL, s.registryJWTSecret, s.registryTokenExpiry)

	r.Get("/healthz", h.Health)

	r.Route("/api/v1", func(r chi.Router) {
		// Öffentliche Routen
		r.Post("/auth/register", h.Register)
		r.Post("/auth/accept-invite", h.AcceptInvite)

		// Auth-geschützte Routen
		r.Group(func(r chi.Router) {
			if s.devMode {
				// Dev-Mode: Fake-Auth für alle geschützten Routen
				r.Use(s.devAuthMiddleware())
			} else {
				// Production: Echte API-Key Authentifizierung
				r.Use(auth.Middleware(s.logger, s.authStore))
			}

			r.Get("/services", h.ListServices)
			r.Post("/services", h.CreateService)
			r.Get("/services/{id}", h.GetService)
			r.Get("/services/{id}/logs", h.StreamLogs)
			r.Delete("/services/{id}", h.DeleteService)

			r.Post("/auth/api-keys", h.CreateAPIKey)
			r.Get("/auth/api-keys", h.ListAPIKeys)
			r.Delete("/auth/api-keys/{id}", h.DeleteAPIKey)
			r.Get("/auth/status", h.AuthStatus)

			r.Post("/auth/invites", h.CreateInvite)
			r.Get("/auth/invites", h.ListInvites)
			r.Delete("/auth/invites/{id}", h.RevokeInvite)

			r.Get("/registry/token", h.GetRegistryToken)
		})
	})

	return r
}

// devAuthMiddleware setzt Fake-Tenant-Informationen für den Dev-Mode.
func (s *Server) devAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prüfe ob Authorization Header vorhanden ist (für manuelle Tests)
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer mc_") {
				rawKey := strings.TrimPrefix(authHeader, "Bearer ")
				info, err := s.authStore.ValidateAPIKey(r.Context(), rawKey)
				if err == nil {
					ctx := auth.WithTenant(r.Context(), info.OrgID, info.UserID)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
			// Fallback: Dev-User mit dev-org
			orgID := "dev-org"
			if s.devOrgUID != "" {
				orgID = s.devOrgUID
			}
			ctx := auth.WithTenant(r.Context(), orgID, "dev-user")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
