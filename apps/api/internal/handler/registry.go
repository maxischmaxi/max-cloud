package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/max-cloud/api/internal/auth"
	"github.com/max-cloud/shared/pkg/models"
)

func (h *Handler) GetRegistryToken(w http.ResponseWriter, r *http.Request) {
	orgID, hasOrgID := auth.OrgIDFromContext(r.Context())
	if !hasOrgID || orgID == "" {
		errorWithRequestID(w, r, "unauthorized", http.StatusUnauthorized)
		return
	}

	scope := r.URL.Query().Get("scope")
	service := r.URL.Query().Get("service")

	if service == "" {
		service = h.registryURL
	}

	if h.registryJWTSecret == "" {
		h.logger.Error("registry jwt secret not configured")
		errorWithRequestID(w, r, "registry not configured", http.StatusInternalServerError)
		return
	}

	access := h.parseScopeToAccess(scope, orgID)

	if !h.validateAccessScope(access, orgID) {
		errResp := map[string]string{
			"error": "access denied to requested scope",
		}
		if reqID := middleware.GetReqID(r.Context()); reqID != "" {
			errResp["request_id"] = reqID
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	now := time.Now()
	expiry := h.registryTokenExpiry
	if expiry == 0 {
		expiry = 1 * time.Hour
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":    "max-cloud",
		"sub":    orgID,
		"aud":    service,
		"exp":    now.Add(expiry).Unix(),
		"nbf":    now.Unix(),
		"iat":    now.Unix(),
		"access": access,
	})

	tokenString, err := token.SignedString([]byte(h.registryJWTSecret))
	if err != nil {
		h.logger.Error("failed to sign token", "error", err)
		errorWithRequestID(w, r, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := models.RegistryTokenResponse{
		Token:       tokenString,
		AccessToken: tokenString,
		ExpiresIn:   int(expiry.Seconds()),
		IssuedAt:    now.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) parseScopeToAccess(scope string, orgID string) []map[string]interface{} {
	if scope == "" {
		return []map[string]interface{}{
			{
				"type":    "registry",
				"name":    "catalog",
				"actions": []string{"*"},
			},
		}
	}

	var access []map[string]interface{}

	scopes := strings.Split(scope, " ")
	for _, s := range scopes {
		parts := strings.Split(s, ":")
		if len(parts) < 3 {
			continue
		}

		scopeType := parts[0]
		name := parts[1]
		actions := strings.Split(parts[2], ",")

		access = append(access, map[string]interface{}{
			"type":    scopeType,
			"name":    name,
			"actions": actions,
		})
	}

	return access
}

func (h *Handler) validateAccessScope(access []map[string]interface{}, orgID string) bool {
	for _, a := range access {
		scopeType, ok := a["type"].(string)
		if !ok {
			continue
		}

		if scopeType == "repository" {
			name, ok := a["name"].(string)
			if !ok {
				return false
			}

			if !h.isOrgRepository(name, orgID) {
				h.logger.Warn("access denied - repository not owned by org",
					"repository", name,
					"org_id", orgID)
				return false
			}
		}
	}

	return true
}

func (h *Handler) isOrgRepository(name string, orgID string) bool {
	expectedPrefix := fmt.Sprintf("%s/", orgID)
	return strings.HasPrefix(name, expectedPrefix)
}
