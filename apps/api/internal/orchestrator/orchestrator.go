package orchestrator

import (
	"context"
	"errors"
	"io"

	"github.com/max-cloud/shared/pkg/models"
)

// ErrNotFound wird zurückgegeben, wenn die Orchestrator-Ressource nicht existiert.
var ErrNotFound = errors.New("orchestrator: resource not found")

// ErrNoPods wird zurückgegeben, wenn keine laufenden Pods gefunden wurden.
var ErrNoPods = errors.New("orchestrator: no running pods found")

// DeployResult enthält das Ergebnis eines Deploy- oder Status-Aufrufs.
type DeployResult struct {
	Status models.ServiceStatus
	URL    string
}

// LogsOptions konfiguriert das Log-Streaming.
type LogsOptions struct {
	Follow bool
	Tail   int64
}

// Orchestrator definiert die Schnittstelle für Container-Orchestrierung.
type Orchestrator interface {
	// Deploy erstellt oder aktualisiert eine Container-Ressource (idempotent).
	Deploy(ctx context.Context, svc models.Service) (*DeployResult, error)
	// Remove löscht eine Container-Ressource (idempotent, kein Fehler wenn nicht vorhanden).
	Remove(ctx context.Context, svc models.Service) error
	// Status liest den aktuellen Zustand einer Container-Ressource.
	Status(ctx context.Context, svc models.Service) (*DeployResult, error)
	// Logs streamt Container-Logs als zeilenweisen Text.
	Logs(ctx context.Context, svc models.Service, opts LogsOptions) (io.ReadCloser, error)
	// CreateNamespace erstellt einen Kubernetes Namespace für eine Organisation.
	CreateNamespace(ctx context.Context, orgID string) error
	// NamespaceExists prüft ob ein Namespace existiert.
	NamespaceExists(ctx context.Context, orgID string) (bool, error)
}
