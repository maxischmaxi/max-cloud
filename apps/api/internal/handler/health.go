package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/max-cloud/api/internal/store"
	"github.com/max-cloud/shared/pkg/models"
)

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	logger *slog.Logger
	store  *store.Store
}

// New creates a new Handler.
func New(logger *slog.Logger, s *store.Store) *Handler {
	return &Handler{logger: logger, store: s}
}

// Health responds with the server health status.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ListServices returns all services.
func (h *Handler) ListServices(w http.ResponseWriter, r *http.Request) {
	services := h.store.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(services)
}

// CreateService creates a new service.
func (h *Handler) CreateService(w http.ResponseWriter, r *http.Request) {
	var req models.DeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Image == "" {
		http.Error(w, `{"error":"name and image are required"}`, http.StatusBadRequest)
		return
	}

	svc := h.store.Create(req)
	h.logger.Info("service created", "id", svc.ID, "name", svc.Name)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(svc)
}

// GetService returns a single service.
func (h *Handler) GetService(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	svc, ok := h.store.Get(id)
	if !ok {
		http.Error(w, `{"error":"service not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(svc)
}

// DeleteService deletes a service.
func (h *Handler) DeleteService(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !h.store.Delete(id) {
		http.Error(w, `{"error":"service not found"}`, http.StatusNotFound)
		return
	}

	h.logger.Info("service deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}
