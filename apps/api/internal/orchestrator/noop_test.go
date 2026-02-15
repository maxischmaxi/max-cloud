package orchestrator

import (
	"bufio"
	"context"
	"log/slog"
	"testing"

	"github.com/max-cloud/shared/pkg/models"
)

func TestNoopDeploy(t *testing.T) {
	orch := NewNoop(slog.Default())
	result, err := orch.Deploy(context.Background(), models.Service{
		Name:  "myapp",
		Image: "nginx:latest",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != models.ServiceStatusReady {
		t.Fatalf("expected status ready, got %s", result.Status)
	}
	if result.URL != "https://myapp.maxcloud.dev" {
		t.Fatalf("expected URL https://myapp.maxcloud.dev, got %s", result.URL)
	}
}

func TestNoopRemove(t *testing.T) {
	orch := NewNoop(slog.Default())
	err := orch.Remove(context.Background(), models.Service{Name: "myapp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNoopLogs(t *testing.T) {
	orch := NewNoop(slog.Default())
	rc, err := orch.Logs(context.Background(), models.Service{Name: "myapp"}, LogsOptions{Tail: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()

	scanner := bufio.NewScanner(rc)
	count := 0
	for scanner.Scan() {
		count++
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 lines, got %d", count)
	}
}

func TestNoopLogsFollowCancel(t *testing.T) {
	orch := NewNoop(slog.Default())
	ctx, cancel := context.WithCancel(context.Background())

	rc, err := orch.Logs(ctx, models.Service{Name: "myapp"}, LogsOptions{Follow: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()

	scanner := bufio.NewScanner(rc)
	// Erste Zeile lesen
	if !scanner.Scan() {
		t.Fatal("expected at least one line")
	}
	// Abbrechen
	cancel()
	// Scanner sollte sauber beenden
	for scanner.Scan() {
		// drain
	}
}

func TestNoopStatus(t *testing.T) {
	orch := NewNoop(slog.Default())
	result, err := orch.Status(context.Background(), models.Service{Name: "myapp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != models.ServiceStatusReady {
		t.Fatalf("expected status ready, got %s", result.Status)
	}
	if result.URL != "https://myapp.maxcloud.dev" {
		t.Fatalf("expected URL https://myapp.maxcloud.dev, got %s", result.URL)
	}
}
