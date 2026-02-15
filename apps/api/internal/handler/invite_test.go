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

func setupInvite() (*Handler, *store.MemoryStore) {
	s := store.NewMemory()
	orch := orchestrator.NewNoop(slog.Default())
	h := New(slog.Default(), s, s, orch, email.NewMock(), 7*24*time.Hour, true, "registry.local", "test-secret", 1*time.Hour)
	return h, s
}

// registerAdmin erstellt einen Admin-User und gibt User, Org und authentifizierten Context zurück.
func registerAdmin(t *testing.T, s *store.MemoryStore) (models.User, models.Organization, context.Context) {
	t.Helper()
	user, org, _, err := s.Register(context.Background(), "admin@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ctx := auth.WithTenant(context.Background(), org.ID, user.ID)
	return user, org, ctx
}

func TestCreateInviteHandler(t *testing.T) {
	h, s := setupInvite()
	_, _, ctx := registerAdmin(t, s)

	payload := `{"email":"new@example.com","role":"member"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/invites", bytes.NewBufferString(payload))
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.CreateInvite(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.InviteResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Invitation.Email != "new@example.com" {
		t.Fatalf("expected email new@example.com, got %s", resp.Invitation.Email)
	}
	if resp.Invitation.Role != models.OrgRoleMember {
		t.Fatalf("expected role member, got %s", resp.Invitation.Role)
	}
	if resp.Token == "" {
		t.Fatal("expected non-empty token in dev mode")
	}
}

func TestCreateInviteNotAdmin(t *testing.T) {
	h, s := setupInvite()
	ctx := context.Background()

	// Admin registrieren, Member einladen und akzeptieren
	admin, org, _, err := s.Register(ctx, "admin@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expires := time.Now().Add(7 * 24 * time.Hour)
	_, rawToken, err := s.CreateInvite(ctx, org.ID, "member@example.com", models.OrgRoleMember, admin.ID, expires)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	member, _, _, _, err := s.AcceptInvite(ctx, rawToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Member versucht einzuladen
	memberCtx := auth.WithTenant(ctx, org.ID, member.ID)
	payload := `{"email":"other@example.com","role":"member"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/invites", bytes.NewBufferString(payload))
	req = req.WithContext(memberCtx)
	w := httptest.NewRecorder()

	h.CreateInvite(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateInviteAlreadyMember(t *testing.T) {
	h, s := setupInvite()
	_, _, ctx := registerAdmin(t, s)

	// Admin versucht sich selbst einzuladen
	payload := `{"email":"admin@example.com","role":"member"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/invites", bytes.NewBufferString(payload))
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.CreateInvite(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAcceptInviteHandler(t *testing.T) {
	h, s := setupInvite()
	admin, org, ctx := registerAdmin(t, s)

	// Einladung erstellen
	expires := time.Now().Add(7 * 24 * time.Hour)
	_, rawToken, err := s.CreateInvite(context.Background(), org.ID, "new@example.com", models.OrgRoleMember, admin.ID, expires)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = ctx

	// Einladung annehmen (kein Auth nötig)
	payload, _ := json.Marshal(models.AcceptInviteRequest{Token: rawToken})
	req := httptest.NewRequest("POST", "/api/v1/auth/accept-invite", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	h.AcceptInvite(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.AcceptInviteResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.User.Email != "new@example.com" {
		t.Fatalf("expected email new@example.com, got %s", resp.User.Email)
	}
	if resp.Organization.ID != org.ID {
		t.Fatalf("expected org ID %s, got %s", org.ID, resp.Organization.ID)
	}
	if resp.APIKey == "" {
		t.Fatal("expected non-empty API key")
	}
}

func TestAcceptInviteBadToken(t *testing.T) {
	h, _ := setupInvite()

	payload := `{"token":"mci_0000000000000000000000000000000000000000000000000000000000000000"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/accept-invite", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()

	h.AcceptInvite(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAcceptInviteExpired(t *testing.T) {
	h, s := setupInvite()
	admin, org, _ := registerAdmin(t, s)

	// Abgelaufene Einladung
	expires := time.Now().Add(-1 * time.Hour)
	_, rawToken, err := s.CreateInvite(context.Background(), org.ID, "new@example.com", models.OrgRoleMember, admin.ID, expires)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	payload, _ := json.Marshal(models.AcceptInviteRequest{Token: rawToken})
	req := httptest.NewRequest("POST", "/api/v1/auth/accept-invite", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	h.AcceptInvite(w, req)

	if w.Code != http.StatusGone {
		t.Fatalf("expected 410, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListInvitesHandler(t *testing.T) {
	h, s := setupInvite()
	admin, org, ctx := registerAdmin(t, s)

	// Einladung erstellen
	expires := time.Now().Add(7 * 24 * time.Hour)
	if _, _, err := s.CreateInvite(context.Background(), org.ID, "a@example.com", models.OrgRoleMember, admin.ID, expires); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/auth/invites", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.ListInvites(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var invites []models.Invitation
	json.NewDecoder(w.Body).Decode(&invites)
	if len(invites) != 1 {
		t.Fatalf("expected 1 invite, got %d", len(invites))
	}
}

func TestListInvitesNotAdmin(t *testing.T) {
	h, s := setupInvite()
	ctx := context.Background()

	admin, org, _, err := s.Register(ctx, "admin@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expires := time.Now().Add(7 * 24 * time.Hour)
	_, rawToken, err := s.CreateInvite(ctx, org.ID, "member@example.com", models.OrgRoleMember, admin.ID, expires)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	member, _, _, _, err := s.AcceptInvite(ctx, rawToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	memberCtx := auth.WithTenant(ctx, org.ID, member.ID)
	req := httptest.NewRequest("GET", "/api/v1/auth/invites", nil)
	req = req.WithContext(memberCtx)
	w := httptest.NewRecorder()

	h.ListInvites(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestRevokeInviteHandler(t *testing.T) {
	h, s := setupInvite()
	admin, org, ctx := registerAdmin(t, s)

	expires := time.Now().Add(7 * 24 * time.Hour)
	invite, _, err := s.CreateInvite(context.Background(), org.ID, "new@example.com", models.OrgRoleMember, admin.ID, expires)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := chi.NewRouter()
	r.Delete("/api/v1/auth/invites/{id}", h.RevokeInvite)

	req := httptest.NewRequest("DELETE", "/api/v1/auth/invites/"+invite.ID, nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRevokeInviteNotFound(t *testing.T) {
	h, s := setupInvite()
	_, _, ctx := registerAdmin(t, s)

	r := chi.NewRouter()
	r.Delete("/api/v1/auth/invites/{id}", h.RevokeInvite)

	req := httptest.NewRequest("DELETE", "/api/v1/auth/invites/nonexistent", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
