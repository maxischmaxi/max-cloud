package store

import (
	"testing"

	"github.com/max-cloud/shared/pkg/models"
)

func TestCreate(t *testing.T) {
	s := New()
	svc := s.Create(models.DeployRequest{
		Name:  "myapp",
		Image: "nginx:latest",
		EnvVars: map[string]string{"PORT": "8080"},
	})

	if svc.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if svc.Name != "myapp" {
		t.Fatalf("expected name myapp, got %s", svc.Name)
	}
	if svc.Image != "nginx:latest" {
		t.Fatalf("expected image nginx:latest, got %s", svc.Image)
	}
	if svc.Status != models.ServiceStatusReady {
		t.Fatalf("expected status ready, got %s", svc.Status)
	}
	if svc.URL == "" {
		t.Fatal("expected non-empty URL")
	}
	if svc.EnvVars["PORT"] != "8080" {
		t.Fatalf("expected env PORT=8080, got %s", svc.EnvVars["PORT"])
	}
	if svc.CreatedAt.IsZero() {
		t.Fatal("expected non-zero CreatedAt")
	}
}

func TestGet(t *testing.T) {
	s := New()
	created := s.Create(models.DeployRequest{Name: "app", Image: "img"})

	svc, ok := s.Get(created.ID)
	if !ok {
		t.Fatal("expected to find service")
	}
	if svc.ID != created.ID {
		t.Fatalf("expected ID %s, got %s", created.ID, svc.ID)
	}
}

func TestGetNotFound(t *testing.T) {
	s := New()
	_, ok := s.Get("nonexistent")
	if ok {
		t.Fatal("expected not found")
	}
}

func TestList(t *testing.T) {
	s := New()

	if svcs := s.List(); len(svcs) != 0 {
		t.Fatalf("expected empty list, got %d", len(svcs))
	}

	s.Create(models.DeployRequest{Name: "a", Image: "img"})
	s.Create(models.DeployRequest{Name: "b", Image: "img"})

	if svcs := s.List(); len(svcs) != 2 {
		t.Fatalf("expected 2 services, got %d", len(svcs))
	}
}

func TestDelete(t *testing.T) {
	s := New()
	created := s.Create(models.DeployRequest{Name: "app", Image: "img"})

	if !s.Delete(created.ID) {
		t.Fatal("expected delete to succeed")
	}

	_, ok := s.Get(created.ID)
	if ok {
		t.Fatal("expected service to be gone after delete")
	}
}

func TestDeleteNotFound(t *testing.T) {
	s := New()
	if s.Delete("nonexistent") {
		t.Fatal("expected delete to return false for nonexistent")
	}
}
