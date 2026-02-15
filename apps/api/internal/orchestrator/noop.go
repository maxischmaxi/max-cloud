package orchestrator

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/max-cloud/shared/pkg/models"
)

// NoopOrchestrator gibt sofort ready zurück — für lokale Entwicklung ohne Cluster.
type NoopOrchestrator struct {
	logger *slog.Logger
}

// NewNoop erstellt einen neuen NoopOrchestrator.
func NewNoop(logger *slog.Logger) *NoopOrchestrator {
	return &NoopOrchestrator{logger: logger}
}

func (n *NoopOrchestrator) Deploy(_ context.Context, svc models.Service) (*DeployResult, error) {
	n.logger.Info("noop: deploy", "name", svc.Name, "image", svc.Image)
	return &DeployResult{
		Status: models.ServiceStatusReady,
		URL:    fmt.Sprintf("https://%s.maxcloud.dev", svc.Name),
	}, nil
}

func (n *NoopOrchestrator) Remove(_ context.Context, svc models.Service) error {
	n.logger.Info("noop: remove", "name", svc.Name)
	return nil
}

func (n *NoopOrchestrator) Status(_ context.Context, svc models.Service) (*DeployResult, error) {
	return &DeployResult{
		Status: models.ServiceStatusReady,
		URL:    fmt.Sprintf("https://%s.maxcloud.dev", svc.Name),
	}, nil
}

func (n *NoopOrchestrator) Logs(ctx context.Context, svc models.Service, opts LogsOptions) (io.ReadCloser, error) {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		tail := opts.Tail
		if tail <= 0 {
			tail = 100
		}

		if !opts.Follow {
			for i := int64(0); i < tail; i++ {
				line := fmt.Sprintf("%s [stdout] noop log line %d for %s\n",
					time.Now().Format(time.RFC3339), i+1, svc.Name)
				if _, err := pw.Write([]byte(line)); err != nil {
					return
				}
			}
			return
		}

		// Follow-Modus: eine Zeile pro Sekunde bis ctx abgebrochen wird
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		i := int64(0)
		for {
			i++
			line := fmt.Sprintf("%s [stdout] noop log line %d for %s\n",
				time.Now().Format(time.RFC3339), i, svc.Name)
			if _, err := pw.Write([]byte(line)); err != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	return pr, nil
}

func (n *NoopOrchestrator) CreateNamespace(_ context.Context, orgID string) error {
	n.logger.Info("noop: create namespace", "org_id", orgID)
	return nil
}

func (n *NoopOrchestrator) NamespaceExists(_ context.Context, orgID string) (bool, error) {
	n.logger.Info("noop: namespace exists check", "org_id", orgID)
	return true, nil
}
