package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/max-cloud/api/internal/config"
	"github.com/max-cloud/api/internal/handler"
	"github.com/max-cloud/api/internal/store"
)

// Server holds dependencies for the API server.
type Server struct {
	cfg    *config.Config
	logger *slog.Logger
}

// New creates a new Server.
func New(cfg *config.Config, logger *slog.Logger) *Server {
	return &Server{cfg: cfg, logger: logger}
}

// Router returns the configured HTTP router.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	st := store.New()
	h := handler.New(s.logger, st)

	r.Get("/healthz", h.Health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/services", h.ListServices)
		r.Post("/services", h.CreateService)
		r.Get("/services/{id}", h.GetService)
		r.Delete("/services/{id}", h.DeleteService)
	})

	return r
}
