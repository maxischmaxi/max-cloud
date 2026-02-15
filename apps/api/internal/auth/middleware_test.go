package auth

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/max-cloud/shared/pkg/models"
)

type mockValidator struct {
	key *models.APIKeyInfo
	err error
}

func (m *mockValidator) ValidateAPIKey(_ context.Context, _ string) (*models.APIKeyInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.key, nil
}

func (m *mockValidator) UpdateAPIKeyLastUsed(_ context.Context, _ string) error {
	return nil
}

func testHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := OrgIDFromContext(r.Context())
		w.Write([]byte("ok:" + orgID))
	})
}

func TestMiddlewareValidKey(t *testing.T) {
	v := &mockValidator{
		key: &models.APIKeyInfo{
			ID:        "key-1",
			OrgID:     "org-1",
			UserID:    "user-1",
			CreatedAt: time.Now(),
		},
	}

	mw := Middleware(slog.Default(), v)
	handler := mw(testHandler())

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer mc_abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "ok:org-1" {
		t.Fatalf("expected ok:org-1, got %s", w.Body.String())
	}
}

func TestMiddlewareMissingHeader(t *testing.T) {
	v := &mockValidator{}
	mw := Middleware(slog.Default(), v)
	handler := mw(testHandler())

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestMiddlewareInvalidKey(t *testing.T) {
	v := &mockValidator{
		err: errKeyNotFound,
	}
	mw := Middleware(slog.Default(), v)
	handler := mw(testHandler())

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer mc_0000000000000000000000000000000000000000000000000000000000000000")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestMiddlewareBadFormat(t *testing.T) {
	v := &mockValidator{}
	mw := Middleware(slog.Default(), v)
	handler := mw(testHandler())

	// Kein "Bearer " Prefix
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Token abc123")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestMiddlewareNotMcPrefix(t *testing.T) {
	v := &mockValidator{}
	mw := Middleware(slog.Default(), v)
	handler := mw(testHandler())

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer sk_1234567890")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// errKeyNotFound ist ein Sentinel-Error für Tests.
var errKeyNotFound = http.ErrAbortHandler // Beliebiger Error als Platzhalter
