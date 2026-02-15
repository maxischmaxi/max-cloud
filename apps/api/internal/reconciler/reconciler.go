package reconciler

import (
	"context"
	"log/slog"
	"time"

	"github.com/max-cloud/api/internal/orchestrator"
	"github.com/max-cloud/api/internal/store"
	"github.com/max-cloud/shared/pkg/models"
)

// Reconciler gleicht den Soll-Zustand (Store) mit dem Ist-Zustand (Orchestrator) ab.
type Reconciler struct {
	logger       *slog.Logger
	store        store.ServiceStore
	orchestrator orchestrator.Orchestrator
	interval     time.Duration
}

// New erstellt einen neuen Reconciler.
func New(logger *slog.Logger, st store.ServiceStore, orch orchestrator.Orchestrator, interval time.Duration) *Reconciler {
	return &Reconciler{
		logger:       logger,
		store:        st,
		orchestrator: orch,
		interval:     interval,
	}
}

// Run startet die Reconcile-Schleife und blockiert bis ctx abgebrochen wird.
func (r *Reconciler) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("reconciler stopped")
			return
		case <-ticker.C:
			r.RunOnce(ctx)
		}
	}
}

// RunOnce führt einen einzelnen Reconcile-Durchlauf aus.
func (r *Reconciler) RunOnce(ctx context.Context) {
	services, err := r.store.List(ctx)
	if err != nil {
		r.logger.Error("reconciler: failed to list services", "error", err)
		return
	}

	for _, svc := range services {
		switch svc.Status {
		case models.ServiceStatusPending:
			r.reconcilePending(ctx, svc)
		case models.ServiceStatusDeleting:
			r.reconcileDeleting(ctx, svc)
		}
	}
}

func (r *Reconciler) reconcilePending(ctx context.Context, svc models.Service) {
	result, err := r.orchestrator.Status(ctx, svc)
	if err != nil {
		if err == orchestrator.ErrNotFound {
			if _, err := r.orchestrator.Deploy(ctx, svc); err != nil {
				r.logger.Error("reconciler: deploy failed", "error", err, "id", svc.ID)
			} else {
				r.logger.Info("reconciler: deployed to knative", "id", svc.ID)
			}
		} else {
			r.logger.Error("reconciler: status check failed", "error", err, "id", svc.ID)
		}
		return
	}

	if result.Status != svc.Status || result.URL != svc.URL {
		if err := r.store.UpdateStatus(ctx, svc.ID, result.Status, result.URL); err != nil {
			r.logger.Error("reconciler: update status failed", "error", err, "id", svc.ID)
			return
		}
		r.logger.Info("reconciler: status updated", "id", svc.ID, "status", result.Status, "url", result.URL)
	}
}

func (r *Reconciler) reconcileDeleting(ctx context.Context, svc models.Service) {
	if err := r.orchestrator.Remove(ctx, svc); err != nil {
		r.logger.Error("reconciler: remove failed", "error", err, "id", svc.ID)
		return
	}

	if err := r.store.Delete(ctx, svc.ID); err != nil {
		r.logger.Error("reconciler: delete from store failed", "error", err, "id", svc.ID)
		return
	}

	r.logger.Info("reconciler: service deleted", "id", svc.ID)
}
