package store

import (
	"context"
	"errors"
	"testing"

	"github.com/max-cloud/api/internal/auth"
	"github.com/max-cloud/shared/pkg/models"
)

func TestCreate(t *testing.T) {
	s := NewMemory()
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
	if svc.URL != "" {
		t.Fatalf("expected empty URL, got %s", svc.URL)
	}
	if svc.EnvVars["PORT"] != "8080" {
		t.Fatalf("expected env PORT=8080, got %s", svc.EnvVars["PORT"])
	}
	if svc.CreatedAt.IsZero() {
		t.Fatal("expected non-zero CreatedAt")
	}
}

func TestGet(t *testing.T) {
	s := NewMemory()
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

func TestGetNotFound(t *testing.T) {
	s := NewMemory()
	_, err := s.Get(context.Background(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestList(t *testing.T) {
	s := NewMemory()
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

func TestDelete(t *testing.T) {
	s := NewMemory()
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

func TestDeleteNotFound(t *testing.T) {
	s := NewMemory()
	err := s.Delete(context.Background(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateStatus(t *testing.T) {
	s := NewMemory()
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

func TestUpdateStatusNotFound(t *testing.T) {
	s := NewMemory()
	err := s.UpdateStatus(context.Background(), "nonexistent", models.ServiceStatusReady, "")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListTenantIsolation(t *testing.T) {
	s := NewMemory()

	ctxOrg1 := auth.WithTenant(context.Background(), "org-1", "user-1")
	ctxOrg2 := auth.WithTenant(context.Background(), "org-2", "user-2")

	if _, err := s.Create(ctxOrg1, models.DeployRequest{Name: "app1", Image: "img"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := s.Create(ctxOrg2, models.DeployRequest{Name: "app2", Image: "img"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Org1 sieht nur eigene Services
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

	// Org2 sieht nur eigene Services
	svcs, err = s.List(ctxOrg2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svcs) != 1 {
		t.Fatalf("expected 1 service for org2, got %d", len(svcs))
	}
	if svcs[0].Name != "app2" {
		t.Fatalf("expected app2, got %s", svcs[0].Name)
	}
}

func TestGetTenantIsolation(t *testing.T) {
	s := NewMemory()

	ctxOrg1 := auth.WithTenant(context.Background(), "org-1", "user-1")
	ctxOrg2 := auth.WithTenant(context.Background(), "org-2", "user-2")

	svc, err := s.Create(ctxOrg1, models.DeployRequest{Name: "app1", Image: "img"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Org1 kann eigenen Service lesen
	if _, err := s.Get(ctxOrg1, svc.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Org2 kann Service von Org1 nicht lesen
	_, err = s.Get(ctxOrg2, svc.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for cross-tenant get, got %v", err)
	}
}

func TestDeleteTenantIsolation(t *testing.T) {
	s := NewMemory()

	ctxOrg1 := auth.WithTenant(context.Background(), "org-1", "user-1")
	ctxOrg2 := auth.WithTenant(context.Background(), "org-2", "user-2")

	svc, err := s.Create(ctxOrg1, models.DeployRequest{Name: "app1", Image: "img"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Org2 kann Service von Org1 nicht löschen
	err = s.Delete(ctxOrg2, svc.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for cross-tenant delete, got %v", err)
	}

	// Org1 kann eigenen Service löschen
	if err := s.Delete(ctxOrg1, svc.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReconcilerSeesAll(t *testing.T) {
	s := NewMemory()

	ctxOrg1 := auth.WithTenant(context.Background(), "org-1", "user-1")
	ctxOrg2 := auth.WithTenant(context.Background(), "org-2", "user-2")
	ctxReconciler := context.Background() // Kein Tenant

	if _, err := s.Create(ctxOrg1, models.DeployRequest{Name: "app1", Image: "img"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := s.Create(ctxOrg2, models.DeployRequest{Name: "app2", Image: "img"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reconciler (kein Tenant) sieht alle Services
	svcs, err := s.List(ctxReconciler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svcs) != 2 {
		t.Fatalf("expected 2 services for reconciler, got %d", len(svcs))
	}
}
