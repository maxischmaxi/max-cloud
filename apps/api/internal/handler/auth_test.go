package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/max-cloud/api/internal/auth"
	"github.com/max-cloud/api/internal/email"
	"github.com/max-cloud/api/internal/orchestrator"
	"github.com/max-cloud/api/internal/store"
	"github.com/max-cloud/shared/pkg/models"
)

func setupAuth() (*Handler, *store.MemoryStore) {
	s := store.NewMemory()
	orch := orchestrator.NewNoop(slog.Default())
	h := New(slog.Default(), s, s, orch, email.NewMock(), 7*24*time.Hour, true, "registry.local", "test-secret", 1*time.Hour)
	return h, s
}

func TestRegisterHandler(t *testing.T) {
	h, _ := setupAuth()

	payload := `{"email":"test@example.com","org_name":"TestOrg"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.RegisterResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.User.Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %s", resp.User.Email)
	}
	if resp.Organization.Name != "TestOrg" {
		t.Fatalf("expected org TestOrg, got %s", resp.Organization.Name)
	}
	if resp.APIKey == "" {
		t.Fatal("expected non-empty API key")
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	h, _ := setupAuth()

	payload := `{"email":"test@example.com","org_name":"Org1"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()
	h.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for first register, got %d", w.Code)
	}

	payload = `{"email":"test@example.com","org_name":"Org2"}`
	req = httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(payload))
	w = httptest.NewRecorder()
	h.Register(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestRegisterValidation(t *testing.T) {
	h, _ := setupAuth()

	tests := []struct {
		name    string
		payload string
	}{
		{"missing email", `{"org_name":"Org1"}`},
		{"missing org_name", `{"email":"test@example.com"}`},
		{"invalid json", `{invalid`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(tt.payload))
			w := httptest.NewRecorder()
			h.Register(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", w.Code)
			}
		})
	}
}

func TestCreateAPIKeyHandler(t *testing.T) {
	h, s := setupAuth()
	ctx := context.Background()

	user, org, _, err := s.Register(ctx, "test@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	payload := `{"name":"ci-key"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/api-keys", bytes.NewBufferString(payload))
	req = req.WithContext(auth.WithTenant(req.Context(), org.ID, user.ID))
	w := httptest.NewRecorder()

	h.CreateAPIKey(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.CreateAPIKeyResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.APIKey == "" {
		t.Fatal("expected non-empty API key")
	}
	if resp.Info.Name != "ci-key" {
		t.Fatalf("expected name ci-key, got %s", resp.Info.Name)
	}
}

func TestListAPIKeysHandler(t *testing.T) {
	h, s := setupAuth()
	ctx := context.Background()

	user, org, _, err := s.Register(ctx, "test@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/auth/api-keys", nil)
	req = req.WithContext(auth.WithTenant(req.Context(), org.ID, user.ID))
	w := httptest.NewRecorder()

	h.ListAPIKeys(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var keys []models.APIKeyInfo
	json.NewDecoder(w.Body).Decode(&keys)
	if len(keys) != 1 {
		t.Fatalf("expected 1 key (default), got %d", len(keys))
	}
}

func TestDeleteAPIKeyHandler(t *testing.T) {
	h, s := setupAuth()
	ctx := context.Background()

	user, org, _, err := s.Register(ctx, "test@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, info, err := s.CreateAPIKey(ctx, org.ID, user.ID, "to-delete")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := chi.NewRouter()
	r.Delete("/api/v1/auth/api-keys/{id}", h.DeleteAPIKey)

	req := httptest.NewRequest("DELETE", "/api/v1/auth/api-keys/"+info.ID, nil)
	req = req.WithContext(auth.WithTenant(req.Context(), org.ID, user.ID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteAPIKeyNotFound(t *testing.T) {
	h, s := setupAuth()
	ctx := context.Background()

	user, org, _, err := s.Register(ctx, "test@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := chi.NewRouter()
	r.Delete("/api/v1/auth/api-keys/{id}", h.DeleteAPIKey)

	req := httptest.NewRequest("DELETE", "/api/v1/auth/api-keys/nonexistent", nil)
	req = req.WithContext(auth.WithTenant(req.Context(), org.ID, user.ID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAuthStatusHandler(t *testing.T) {
	h, s := setupAuth()
	ctx := context.Background()

	user, org, _, err := s.Register(ctx, "test@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/auth/status", nil)
	req = req.WithContext(auth.WithTenant(req.Context(), org.ID, user.ID))
	w := httptest.NewRecorder()

	h.AuthStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var info models.AuthInfo
	json.NewDecoder(w.Body).Decode(&info)
	if info.User.Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %s", info.User.Email)
	}
	if info.Role != "admin" {
		t.Fatalf("expected role admin, got %s", info.Role)
	}
}
