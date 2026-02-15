package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/max-cloud/api/internal/auth"
	"github.com/max-cloud/api/internal/store"
	"github.com/max-cloud/shared/pkg/models"
)

// Register erstellt einen neuen Account (User + Organization + API-Key).
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.OrgName == "" {
		http.Error(w, `{"error":"email and org_name are required"}`, http.StatusBadRequest)
		return
	}

	user, org, rawKey, err := h.authStore.Register(r.Context(), req.Email, req.OrgName)
	if err != nil {
		if errors.Is(err, store.ErrDuplicateEmail) {
			http.Error(w, `{"error":"email already registered"}`, http.StatusConflict)
			return
		}
		if errors.Is(err, store.ErrDuplicateOrg) {
			http.Error(w, `{"error":"organization name already taken"}`, http.StatusConflict)
			return
		}
		h.logger.Error("failed to register", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	h.logger.Info("user registered", "email", req.Email, "org", req.OrgName, "org_id", org.ID)

	resp := models.RegisterResponse{
		User:         user,
		Organization: org,
		APIKey:       rawKey,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// CreateAPIKey erstellt einen neuen API-Key für die aktuelle Organisation.
func (h *Handler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req models.CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
		return
	}

	orgID, _ := auth.OrgIDFromContext(r.Context())
	userID, _ := auth.UserIDFromContext(r.Context())

	rawKey, info, err := h.authStore.CreateAPIKey(r.Context(), orgID, userID, req.Name)
	if err != nil {
		h.logger.Error("failed to create api key", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	resp := models.CreateAPIKeyResponse{
		APIKey: rawKey,
		Info:   *info,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// ListAPIKeys gibt alle API-Keys der aktuellen Organisation zurück.
func (h *Handler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	orgID, _ := auth.OrgIDFromContext(r.Context())

	keys, err := h.authStore.ListAPIKeys(r.Context(), orgID)
	if err != nil {
		h.logger.Error("failed to list api keys", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}

// DeleteAPIKey löscht einen API-Key.
func (h *Handler) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	keyID := chi.URLParam(r, "id")
	orgID, _ := auth.OrgIDFromContext(r.Context())

	if err := h.authStore.DeleteAPIKey(r.Context(), orgID, keyID); err != nil {
		if errors.Is(err, store.ErrKeyNotFound) {
			http.Error(w, `{"error":"api key not found"}`, http.StatusNotFound)
			return
		}
		h.logger.Error("failed to delete api key", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AuthStatus gibt Informationen über den aktuellen Benutzer zurück.
func (h *Handler) AuthStatus(w http.ResponseWriter, r *http.Request) {
	orgID, _ := auth.OrgIDFromContext(r.Context())
	userID, _ := auth.UserIDFromContext(r.Context())

	info, err := h.authStore.GetAuthInfo(r.Context(), orgID, userID)
	if err != nil {
		h.logger.Error("failed to get auth info", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}
