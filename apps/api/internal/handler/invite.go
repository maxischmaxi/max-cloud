package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/max-cloud/api/internal/auth"
	"github.com/max-cloud/api/internal/store"
	"github.com/max-cloud/shared/pkg/models"
)

// CreateInvite erstellt eine neue Einladung (nur für Admins).
func (h *Handler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	orgID, _ := auth.OrgIDFromContext(r.Context())
	userID, _ := auth.UserIDFromContext(r.Context())

	// Admin-Check
	info, err := h.authStore.GetAuthInfo(r.Context(), orgID, userID)
	if err != nil {
		h.logger.Error("failed to get auth info", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	if info.Role != models.OrgRoleAdmin {
		http.Error(w, `{"error":"admin role required"}`, http.StatusForbidden)
		return
	}

	var req models.InviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, `{"error":"email is required"}`, http.StatusBadRequest)
		return
	}

	if req.Role == "" {
		req.Role = models.OrgRoleMember
	}
	if req.Role != models.OrgRoleAdmin && req.Role != models.OrgRoleMember {
		http.Error(w, `{"error":"role must be admin or member"}`, http.StatusBadRequest)
		return
	}

	expiresAt := time.Now().Add(h.inviteExpiry)
	invite, rawToken, err := h.authStore.CreateInvite(r.Context(), orgID, req.Email, req.Role, userID, expiresAt)
	if err != nil {
		if errors.Is(err, store.ErrAlreadyMember) {
			http.Error(w, `{"error":"user is already a member of this organization"}`, http.StatusConflict)
			return
		}
		h.logger.Error("failed to create invite", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	// E-Mail senden
	if err := h.emailSender.SendInvite(r.Context(), req.Email, info.Organization.Name, rawToken); err != nil {
		h.logger.Error("failed to send invite email", "error", err, "email", req.Email)
		http.Error(w, `{"error":"failed to send invite email"}`, http.StatusInternalServerError)
		return
	}

	resp := models.InviteResponse{
		Invitation: invite,
	}
	if h.devMode {
		resp.Token = rawToken
	}

	h.logger.Info("invite created and email sent", "email", req.Email, "org", orgID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// ListInvites gibt alle pending Einladungen der aktuellen Org zurück (nur für Admins).
func (h *Handler) ListInvites(w http.ResponseWriter, r *http.Request) {
	orgID, _ := auth.OrgIDFromContext(r.Context())
	userID, _ := auth.UserIDFromContext(r.Context())

	info, err := h.authStore.GetAuthInfo(r.Context(), orgID, userID)
	if err != nil {
		h.logger.Error("failed to get auth info", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	if info.Role != models.OrgRoleAdmin {
		http.Error(w, `{"error":"admin role required"}`, http.StatusForbidden)
		return
	}

	invites, err := h.authStore.ListInvites(r.Context(), orgID)
	if err != nil {
		h.logger.Error("failed to list invites", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invites)
}

// RevokeInvite widerruft eine Einladung (nur für Admins).
func (h *Handler) RevokeInvite(w http.ResponseWriter, r *http.Request) {
	orgID, _ := auth.OrgIDFromContext(r.Context())
	userID, _ := auth.UserIDFromContext(r.Context())
	inviteID := chi.URLParam(r, "id")

	info, err := h.authStore.GetAuthInfo(r.Context(), orgID, userID)
	if err != nil {
		h.logger.Error("failed to get auth info", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	if info.Role != models.OrgRoleAdmin {
		http.Error(w, `{"error":"admin role required"}`, http.StatusForbidden)
		return
	}

	if err := h.authStore.RevokeInvite(r.Context(), orgID, inviteID); err != nil {
		if errors.Is(err, store.ErrInviteNotFound) {
			http.Error(w, `{"error":"invite not found"}`, http.StatusNotFound)
			return
		}
		h.logger.Error("failed to revoke invite", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AcceptInvite nimmt eine Einladung an (öffentlich, kein Auth nötig).
func (h *Handler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	var req models.AcceptInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		http.Error(w, `{"error":"token is required"}`, http.StatusBadRequest)
		return
	}

	user, org, role, rawAPIKey, err := h.authStore.AcceptInvite(r.Context(), req.Token)
	if err != nil {
		if errors.Is(err, store.ErrInviteNotFound) {
			http.Error(w, `{"error":"invite not found"}`, http.StatusNotFound)
			return
		}
		if errors.Is(err, store.ErrInviteExpired) {
			http.Error(w, `{"error":"invite expired"}`, http.StatusGone)
			return
		}
		h.logger.Error("failed to accept invite", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	h.logger.Info("invite accepted", "email", user.Email, "org", org.ID)

	resp := models.AcceptInviteResponse{
		User:         user,
		Organization: org,
		Role:         role,
		APIKey:       rawAPIKey,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
