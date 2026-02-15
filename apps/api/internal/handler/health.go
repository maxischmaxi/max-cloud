package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/max-cloud/api/internal/email"
	"github.com/max-cloud/api/internal/orchestrator"
	"github.com/max-cloud/api/internal/store"
	"github.com/max-cloud/shared/pkg/models"
)

type Handler struct {
	logger              *slog.Logger
	store               store.ServiceStore
	authStore           store.AuthStore
	orchestrator        orchestrator.Orchestrator
	emailSender         email.Sender
	inviteExpiry        time.Duration
	devMode             bool
	registryURL         string
	registryJWTSecret   string
	registryTokenExpiry time.Duration
}

func New(logger *slog.Logger, st store.ServiceStore, authSt store.AuthStore, orch orchestrator.Orchestrator, emailSender email.Sender, inviteExpiry time.Duration, devMode bool, registryURL string, registryJWTSecret string, registryTokenExpiry time.Duration) *Handler {
	return &Handler{
		logger:              logger,
		store:               st,
		authStore:           authSt,
		orchestrator:        orch,
		emailSender:         emailSender,
		inviteExpiry:        inviteExpiry,
		devMode:             devMode,
		registryURL:         registryURL,
		registryJWTSecret:   registryJWTSecret,
		registryTokenExpiry: registryTokenExpiry,
	}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func errorWithRequestID(w http.ResponseWriter, r *http.Request, msg string, code int) {
	errResp := map[string]string{"error": msg}
	if reqID := middleware.GetReqID(r.Context()); reqID != "" {
		errResp["request_id"] = reqID
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(errResp)
}

func (h *Handler) CreateService(w http.ResponseWriter, r *http.Request) {
	var req models.DeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("invalid request body", "error", err)
		errorWithRequestID(w, r, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Image == "" {
		errorWithRequestID(w, r, "name and image are required", http.StatusBadRequest)
		return
	}

	svc, err := h.store.Create(r.Context(), req)
	if err != nil {
		if errors.Is(err, store.ErrDuplicateService) {
			errorWithRequestID(w, r, "service with this name already exists", http.StatusConflict)
			return
		}
		h.logger.Error("failed to create service", "error", err, "name", req.Name)
		errorWithRequestID(w, r, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(svc)
}

func (h *Handler) ListServices(w http.ResponseWriter, r *http.Request) {
	services, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("failed to list services", "error", err)
		errorWithRequestID(w, r, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(services)
}

func (h *Handler) GetService(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	svc, err := h.store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, `{"error":"service not found"}`, http.StatusNotFound)
			return
		}
		h.logger.Error("failed to get service", "error", err, "id", id)
		errorWithRequestID(w, r, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(svc)
}

func (h *Handler) DeleteService(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	svc, err := h.store.Get(r.Context(), id)
	if err != nil {
		if !isUUIDError(err) && !errors.Is(err, store.ErrNotFound) {
			h.logger.Error("failed to get service for delete", "error", err, "id", id, "request_id", middleware.GetReqID(r.Context()))
			errorWithRequestID(w, r, "internal server error", http.StatusInternalServerError)
			return
		}
		svcByName, err := h.store.GetByName(r.Context(), id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				http.Error(w, `{"error":"service not found"}`, http.StatusNotFound)
				return
			}
			h.logger.Error("failed to get service by name for delete", "error", err, "name", id)
			errorWithRequestID(w, r, "internal server error", http.StatusInternalServerError)
			return
		}
		svc = svcByName
	}

	if h.orchestrator != nil {
		if err := h.orchestrator.Remove(r.Context(), svc); err != nil {
			h.logger.Error("orchestrator remove failed", "error", err, "id", svc.ID)
		}
	}

	if err := h.store.Delete(r.Context(), svc.ID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, `{"error":"service not found"}`, http.StatusNotFound)
			return
		}
		h.logger.Error("failed to delete service", "error", err, "id", svc.ID, "request_id", middleware.GetReqID(r.Context()))
		errorWithRequestID(w, r, "internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("service deleted", "id", svc.ID)
	w.WriteHeader(http.StatusNoContent)
}

func isUUIDError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "invalid input syntax for type uuid")
}
