package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/max-cloud/shared/pkg/models"
)

func TestRegister(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	user, org, rawKey, err := s.Register(ctx, "test@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.ID == "" {
		t.Fatal("expected non-empty user ID")
	}
	if user.Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %s", user.Email)
	}
	if org.ID == "" {
		t.Fatal("expected non-empty org ID")
	}
	if org.Name != "TestOrg" {
		t.Fatalf("expected org name TestOrg, got %s", org.Name)
	}
	if !strings.HasPrefix(rawKey, "mc_") {
		t.Fatalf("expected key to start with mc_, got %s", rawKey)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	if _, _, _, err := s.Register(ctx, "test@example.com", "Org1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, _, _, err := s.Register(ctx, "test@example.com", "Org2")
	if err != ErrDuplicateEmail {
		t.Fatalf("expected ErrDuplicateEmail, got %v", err)
	}
}

func TestRegisterDuplicateOrg(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	if _, _, _, err := s.Register(ctx, "a@example.com", "SameOrg"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, _, _, err := s.Register(ctx, "b@example.com", "SameOrg")
	if err != ErrDuplicateOrg {
		t.Fatalf("expected ErrDuplicateOrg, got %v", err)
	}
}

func TestValidateAPIKey(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	_, org, rawKey, err := s.Register(ctx, "test@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := s.ValidateAPIKey(ctx, rawKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.OrgID != org.ID {
		t.Fatalf("expected org ID %s, got %s", org.ID, info.OrgID)
	}
}

func TestValidateAPIKeyInvalid(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	_, err := s.ValidateAPIKey(ctx, "mc_0000000000000000000000000000000000000000000000000000000000000000")
	if err != ErrKeyNotFound {
		t.Fatalf("expected ErrKeyNotFound, got %v", err)
	}
}

func TestValidateAPIKeyBadFormat(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	_, err := s.ValidateAPIKey(ctx, "bad_key")
	if err != ErrKeyNotFound {
		t.Fatalf("expected ErrKeyNotFound, got %v", err)
	}
}

func TestCreateAPIKey(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	user, org, _, err := s.Register(ctx, "test@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rawKey, info, err := s.CreateAPIKey(ctx, org.ID, user.ID, "ci-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(rawKey, "mc_") {
		t.Fatalf("expected key to start with mc_, got %s", rawKey)
	}
	if info.Name != "ci-key" {
		t.Fatalf("expected name ci-key, got %s", info.Name)
	}

	// Validierung des neuen Keys
	validated, err := s.ValidateAPIKey(ctx, rawKey)
	if err != nil {
		t.Fatalf("unexpected error validating new key: %v", err)
	}
	if validated.ID != info.ID {
		t.Fatalf("expected key ID %s, got %s", info.ID, validated.ID)
	}
}

func TestListAPIKeys(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	user, org, _, err := s.Register(ctx, "test@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Sollte bereits 1 Key haben (default von Register)
	keys, err := s.ListAPIKeys(ctx, org.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}

	// Zweiten Key erstellen
	if _, _, err := s.CreateAPIKey(ctx, org.ID, user.ID, "second"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	keys, err = s.ListAPIKeys(ctx, org.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}

func TestDeleteAPIKey(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	user, org, _, err := s.Register(ctx, "test@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, info, err := s.CreateAPIKey(ctx, org.ID, user.ID, "to-delete")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := s.DeleteAPIKey(ctx, org.ID, info.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Erneutes Löschen sollte ErrKeyNotFound geben
	if err := s.DeleteAPIKey(ctx, org.ID, info.ID); err != ErrKeyNotFound {
		t.Fatalf("expected ErrKeyNotFound, got %v", err)
	}
}

func TestGetAuthInfo(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	user, org, _, err := s.Register(ctx, "test@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := s.GetAuthInfo(ctx, org.ID, user.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.User.Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %s", info.User.Email)
	}
	if info.Organization.Name != "TestOrg" {
		t.Fatalf("expected org TestOrg, got %s", info.Organization.Name)
	}
	if info.Role != "admin" {
		t.Fatalf("expected role admin, got %s", info.Role)
	}
}

func TestCreateInvite(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	user, org, _, err := s.Register(ctx, "admin@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expires := time.Now().Add(7 * 24 * time.Hour)
	invite, rawToken, err := s.CreateInvite(ctx, org.ID, "new@example.com", models.OrgRoleMember, user.ID, expires)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if invite.ID == "" {
		t.Fatal("expected non-empty invite ID")
	}
	if invite.Email != "new@example.com" {
		t.Fatalf("expected email new@example.com, got %s", invite.Email)
	}
	if invite.Role != models.OrgRoleMember {
		t.Fatalf("expected role member, got %s", invite.Role)
	}
	if invite.Status != models.InviteStatusPending {
		t.Fatalf("expected status pending, got %s", invite.Status)
	}
	if !strings.HasPrefix(rawToken, "mci_") {
		t.Fatalf("expected token to start with mci_, got %s", rawToken)
	}
}

func TestCreateInviteAlreadyMember(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	user, org, _, err := s.Register(ctx, "admin@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expires := time.Now().Add(7 * 24 * time.Hour)
	_, _, err = s.CreateInvite(ctx, org.ID, "admin@example.com", models.OrgRoleMember, user.ID, expires)
	if err != ErrAlreadyMember {
		t.Fatalf("expected ErrAlreadyMember, got %v", err)
	}
}

func TestAcceptInvite(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	admin, org, _, err := s.Register(ctx, "admin@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expires := time.Now().Add(7 * 24 * time.Hour)
	_, rawToken, err := s.CreateInvite(ctx, org.ID, "new@example.com", models.OrgRoleMember, admin.ID, expires)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	user, retOrg, role, apiKey, err := s.AcceptInvite(ctx, rawToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.Email != "new@example.com" {
		t.Fatalf("expected email new@example.com, got %s", user.Email)
	}
	if retOrg.ID != org.ID {
		t.Fatalf("expected org ID %s, got %s", org.ID, retOrg.ID)
	}
	if role != models.OrgRoleMember {
		t.Fatalf("expected role member, got %s", role)
	}
	if !strings.HasPrefix(apiKey, "mc_") {
		t.Fatalf("expected api key to start with mc_, got %s", apiKey)
	}
}

func TestAcceptInviteExistingUser(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	// Zwei Orgs registrieren
	admin1, org1, _, err := s.Register(ctx, "admin@example.com", "Org1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, _, _, err = s.Register(ctx, "existing@example.com", "Org2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// existing@example.com zu Org1 einladen
	expires := time.Now().Add(7 * 24 * time.Hour)
	_, rawToken, err := s.CreateInvite(ctx, org1.ID, "existing@example.com", models.OrgRoleMember, admin1.ID, expires)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	user, retOrg, role, _, err := s.AcceptInvite(ctx, rawToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.Email != "existing@example.com" {
		t.Fatalf("expected email existing@example.com, got %s", user.Email)
	}
	if retOrg.ID != org1.ID {
		t.Fatalf("expected org ID %s, got %s", org1.ID, retOrg.ID)
	}
	if role != models.OrgRoleMember {
		t.Fatalf("expected role member, got %s", role)
	}
}

func TestAcceptInviteExpired(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	admin, org, _, err := s.Register(ctx, "admin@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Einladung die bereits abgelaufen ist
	expires := time.Now().Add(-1 * time.Hour)
	_, rawToken, err := s.CreateInvite(ctx, org.ID, "new@example.com", models.OrgRoleMember, admin.ID, expires)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, _, _, _, err = s.AcceptInvite(ctx, rawToken)
	if err != ErrInviteExpired {
		t.Fatalf("expected ErrInviteExpired, got %v", err)
	}
}

func TestAcceptInviteInvalidToken(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	_, _, _, _, err := s.AcceptInvite(ctx, "mci_0000000000000000000000000000000000000000000000000000000000000000")
	if err != ErrInviteNotFound {
		t.Fatalf("expected ErrInviteNotFound, got %v", err)
	}
}

func TestAcceptInviteAlreadyUsed(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	admin, org, _, err := s.Register(ctx, "admin@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expires := time.Now().Add(7 * 24 * time.Hour)
	_, rawToken, err := s.CreateInvite(ctx, org.ID, "new@example.com", models.OrgRoleMember, admin.ID, expires)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Erstes Accept
	if _, _, _, _, err := s.AcceptInvite(ctx, rawToken); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Zweites Accept sollte fehlschlagen
	_, _, _, _, err = s.AcceptInvite(ctx, rawToken)
	if err != ErrInviteNotFound {
		t.Fatalf("expected ErrInviteNotFound for used token, got %v", err)
	}
}

func TestListInvites(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	admin, org, _, err := s.Register(ctx, "admin@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Leere Liste
	invites, err := s.ListInvites(ctx, org.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(invites) != 0 {
		t.Fatalf("expected 0 invites, got %d", len(invites))
	}

	// Einladung erstellen
	expires := time.Now().Add(7 * 24 * time.Hour)
	if _, _, err := s.CreateInvite(ctx, org.ID, "a@example.com", models.OrgRoleMember, admin.ID, expires); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, _, err := s.CreateInvite(ctx, org.ID, "b@example.com", models.OrgRoleAdmin, admin.ID, expires); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	invites, err = s.ListInvites(ctx, org.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(invites) != 2 {
		t.Fatalf("expected 2 invites, got %d", len(invites))
	}
}

func TestRevokeInvite(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	admin, org, _, err := s.Register(ctx, "admin@example.com", "TestOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expires := time.Now().Add(7 * 24 * time.Hour)
	invite, _, err := s.CreateInvite(ctx, org.ID, "new@example.com", models.OrgRoleMember, admin.ID, expires)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := s.RevokeInvite(ctx, org.ID, invite.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Nach Revoke sollte die Einladung nicht mehr in der Liste sein
	invites, err := s.ListInvites(ctx, org.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(invites) != 0 {
		t.Fatalf("expected 0 invites after revoke, got %d", len(invites))
	}

	// Erneutes Revoke sollte fehlschlagen
	if err := s.RevokeInvite(ctx, org.ID, invite.ID); err != ErrInviteNotFound {
		t.Fatalf("expected ErrInviteNotFound, got %v", err)
	}
}
