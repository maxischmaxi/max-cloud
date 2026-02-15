package store

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/max-cloud/api/internal/auth"
	"github.com/max-cloud/shared/pkg/models"
)

func newPostgresStore(t *testing.T) *PostgresStore {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping PostgresStore tests")
	}

	ctx := context.Background()
	s, err := NewPostgres(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to create PostgresStore: %v", err)
	}

	// Tabellen vor jedem Test leeren (Reihenfolge wegen FK-Constraints)
	for _, table := range []string{"invitations", "api_keys", "org_members", "services", "users", "organizations"} {
		if _, err := s.pool.Exec(ctx, "DELETE FROM "+table); err != nil {
			t.Fatalf("failed to clean %s table: %v", table, err)
		}
	}

	t.Cleanup(func() { s.Close() })
	return s
}

func TestPostgresCreate(t *testing.T) {
	s := newPostgresStore(t)
	ctx := context.Background()

	svc, err := s.Create(ctx, models.DeployRequest{
		Name:    "myapp",
		Image:   "nginx:latest",
		EnvVars: map[string]string{"PORT": "8080"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if svc.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if svc.Name != "myapp" {
		t.Fatalf("expected name myapp, got %s", svc.Name)
	}
	if svc.Image != "nginx:latest" {
		t.Fatalf("expected image nginx:latest, got %s", svc.Image)
	}
	if svc.Status != models.ServiceStatusPending {
		t.Fatalf("expected status pending, got %s", svc.Status)
	}
	if svc.EnvVars["PORT"] != "8080" {
		t.Fatalf("expected env PORT=8080, got %s", svc.EnvVars["PORT"])
	}
	if svc.CreatedAt.IsZero() {
		t.Fatal("expected non-zero CreatedAt")
	}
}

func TestPostgresCreateNilEnvVars(t *testing.T) {
	s := newPostgresStore(t)
	ctx := context.Background()

	svc, err := s.Create(ctx, models.DeployRequest{
		Name:  "noenv",
		Image: "nginx:latest",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.EnvVars == nil {
		t.Fatal("expected non-nil EnvVars map")
	}
}

func TestPostgresGet(t *testing.T) {
	s := newPostgresStore(t)
	ctx := context.Background()

	created, err := s.Create(ctx, models.DeployRequest{Name: "app", Image: "img"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc, err := s.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.ID != created.ID {
		t.Fatalf("expected ID %s, got %s", created.ID, svc.ID)
	}
}

func TestPostgresGetNotFound(t *testing.T) {
	s := newPostgresStore(t)
	_, err := s.Get(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPostgresList(t *testing.T) {
	s := newPostgresStore(t)
	ctx := context.Background()

	svcs, err := s.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svcs) != 0 {
		t.Fatalf("expected empty list, got %d", len(svcs))
	}

	if _, err := s.Create(ctx, models.DeployRequest{Name: "a", Image: "img"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := s.Create(ctx, models.DeployRequest{Name: "b", Image: "img"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svcs, err = s.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svcs) != 2 {
		t.Fatalf("expected 2 services, got %d", len(svcs))
	}
}

func TestPostgresDelete(t *testing.T) {
	s := newPostgresStore(t)
	ctx := context.Background()

	created, err := s.Create(ctx, models.DeployRequest{Name: "app", Image: "img"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := s.Delete(ctx, created.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = s.Get(ctx, created.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestPostgresDeleteNotFound(t *testing.T) {
	s := newPostgresStore(t)
	err := s.Delete(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPostgresUpdateStatus(t *testing.T) {
	s := newPostgresStore(t)
	ctx := context.Background()

	created, err := s.Create(ctx, models.DeployRequest{Name: "app", Image: "img"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.Status != models.ServiceStatusPending {
		t.Fatalf("expected pending, got %s", created.Status)
	}

	err = s.UpdateStatus(ctx, created.ID, models.ServiceStatusReady, "https://app.maxcloud.dev")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc, err := s.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.Status != models.ServiceStatusReady {
		t.Fatalf("expected ready, got %s", svc.Status)
	}
	if svc.URL != "https://app.maxcloud.dev" {
		t.Fatalf("expected URL https://app.maxcloud.dev, got %s", svc.URL)
	}
}

func TestPostgresUpdateStatusNotFound(t *testing.T) {
	s := newPostgresStore(t)
	err := s.UpdateStatus(context.Background(), "00000000-0000-0000-0000-000000000000", models.ServiceStatusReady, "")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPostgresListTenantIsolation(t *testing.T) {
	s := newPostgresStore(t)
	ctx := context.Background()

	// Zwei Orgs registrieren
	_, org1, _, err := s.Register(ctx, "a@example.com", "Org1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, org2, _, err := s.Register(ctx, "b@example.com", "Org2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctxOrg1 := auth.WithTenant(ctx, org1.ID, "user-1")
	ctxOrg2 := auth.WithTenant(ctx, org2.ID, "user-2")

	if _, err := s.Create(ctxOrg1, models.DeployRequest{Name: "app1", Image: "img"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := s.Create(ctxOrg2, models.DeployRequest{Name: "app2", Image: "img"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svcs, err := s.List(ctxOrg1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svcs) != 1 {
		t.Fatalf("expected 1 service for org1, got %d", len(svcs))
	}
	if svcs[0].Name != "app1" {
		t.Fatalf("expected app1, got %s", svcs[0].Name)
	}
}

func TestPostgresGetTenantIsolation(t *testing.T) {
	s := newPostgresStore(t)
	ctx := context.Background()

	_, org1, _, err := s.Register(ctx, "a@example.com", "Org1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, org2, _, err := s.Register(ctx, "b@example.com", "Org2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctxOrg1 := auth.WithTenant(ctx, org1.ID, "user-1")
	ctxOrg2 := auth.WithTenant(ctx, org2.ID, "user-2")

	svc, err := s.Create(ctxOrg1, models.DeployRequest{Name: "app1", Image: "img"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := s.Get(ctxOrg1, svc.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = s.Get(ctxOrg2, svc.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for cross-tenant get, got %v", err)
	}
}

func TestPostgresReconcilerSeesAll(t *testing.T) {
	s := newPostgresStore(t)
	ctx := context.Background()

	_, org1, _, err := s.Register(ctx, "a@example.com", "Org1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, org2, _, err := s.Register(ctx, "b@example.com", "Org2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctxOrg1 := auth.WithTenant(ctx, org1.ID, "user-1")
	ctxOrg2 := auth.WithTenant(ctx, org2.ID, "user-2")

	if _, err := s.Create(ctxOrg1, models.DeployRequest{Name: "app1", Image: "img"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := s.Create(ctxOrg2, models.DeployRequest{Name: "app2", Image: "img"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reconciler (kein Tenant im Context) sieht alle
	svcs, err := s.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svcs) != 2 {
		t.Fatalf("expected 2 services for reconciler, got %d", len(svcs))
	}
}
