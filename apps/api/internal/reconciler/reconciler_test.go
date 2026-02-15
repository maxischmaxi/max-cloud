package reconciler

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/max-cloud/api/internal/orchestrator"
	"github.com/max-cloud/api/internal/store"
	"github.com/max-cloud/shared/pkg/models"
)

func TestReconcilePendingToReady(t *testing.T) {
	st := store.NewMemory()
	orch := orchestrator.NewNoop(slog.Default())
	rec := New(slog.Default(), st, orch, time.Second)
	ctx := context.Background()

	svc, err := st.Create(ctx, models.DeployRequest{Name: "myapp", Image: "nginx:latest"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.Status != models.ServiceStatusPending {
		t.Fatalf("expected pending, got %s", svc.Status)
	}

	rec.RunOnce(ctx)

	updated, err := st.Get(ctx, svc.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != models.ServiceStatusReady {
		t.Fatalf("expected ready after reconcile, got %s", updated.Status)
	}
	if updated.URL != "https://myapp.maxcloud.dev" {
		t.Fatalf("expected URL https://myapp.maxcloud.dev, got %s", updated.URL)
	}
}

func TestReconcileDeleting(t *testing.T) {
	st := store.NewMemory()
	orch := orchestrator.NewNoop(slog.Default())
	rec := New(slog.Default(), st, orch, time.Second)
	ctx := context.Background()

	svc, err := st.Create(ctx, models.DeployRequest{Name: "myapp", Image: "nginx:latest"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := st.UpdateStatus(ctx, svc.ID, models.ServiceStatusDeleting, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec.RunOnce(ctx)

	_, err = st.Get(ctx, svc.ID)
	if err == nil {
		t.Fatal("expected service to be deleted after reconcile")
	}
}

func TestReconcileSkipsReady(t *testing.T) {
	st := store.NewMemory()
	orch := orchestrator.NewNoop(slog.Default())
	rec := New(slog.Default(), st, orch, time.Second)
	ctx := context.Background()

	svc, err := st.Create(ctx, models.DeployRequest{Name: "myapp", Image: "nginx:latest"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := st.UpdateStatus(ctx, svc.ID, models.ServiceStatusReady, "https://myapp.maxcloud.dev"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec.RunOnce(ctx)

	updated, err := st.Get(ctx, svc.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != models.ServiceStatusReady {
		t.Fatalf("expected ready, got %s", updated.Status)
	}
}
