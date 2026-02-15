package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/max-cloud/shared/pkg/models"
)

// KeyValidator ist das Interface das die Middleware zum Validieren von API-Keys braucht.
type KeyValidator interface {
	ValidateAPIKey(ctx context.Context, rawKey string) (*models.APIKeyInfo, error)
	UpdateAPIKeyLastUsed(ctx context.Context, keyID string) error
}

// Middleware erstellt eine HTTP-Middleware die API-Key-Authentifizierung erzwingt.
func Middleware(logger *slog.Logger, validator KeyValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			rawKey := strings.TrimPrefix(authHeader, "Bearer ")
			if rawKey == authHeader {
				http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
				return
			}

			if !strings.HasPrefix(rawKey, "mc_") {
				http.Error(w, `{"error":"invalid api key format"}`, http.StatusUnauthorized)
				return
			}

			info, err := validator.ValidateAPIKey(r.Context(), rawKey)
			if err != nil {
				http.Error(w, `{"error":"invalid api key"}`, http.StatusUnauthorized)
				return
			}

			// last_used_at async aktualisieren (Fehler nicht blockierend)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						logger.Error("panic in last_used update", "panic", r, "key_id", info.ID)
					}
				}()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := validator.UpdateAPIKeyLastUsed(ctx, info.ID); err != nil {
					logger.Warn("failed to update api key last_used_at", "error", err, "key_id", info.ID)
				}
			}()

			ctx := WithTenant(r.Context(), info.OrgID, info.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
