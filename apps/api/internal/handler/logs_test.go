package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/max-cloud/api/internal/email"
	"github.com/max-cloud/api/internal/orchestrator"
	"github.com/max-cloud/api/internal/store"
	"github.com/max-cloud/shared/pkg/models"
)

// mockOrchestrator implementiert das volle Orchestrator-Interface für Handler-Tests.
type mockOrchestrator struct {
	logsReader io.ReadCloser
	logsErr    error
}

func (m *mockOrchestrator) Deploy(_ context.Context, _ models.Service) (*orchestrator.DeployResult, error) {
	return &orchestrator.DeployResult{Status: models.ServiceStatusReady, URL: "https://test.maxcloud.dev"}, nil
}

func (m *mockOrchestrator) Remove(_ context.Context, _ models.Service) error {
	return nil
}

func (m *mockOrchestrator) Status(_ context.Context, _ models.Service) (*orchestrator.DeployResult, error) {
	return &orchestrator.DeployResult{Status: models.ServiceStatusReady}, nil
}

func (m *mockOrchestrator) Logs(_ context.Context, _ models.Service, _ orchestrator.LogsOptions) (io.ReadCloser, error) {
	if m.logsErr != nil {
		return nil, m.logsErr
	}
	return m.logsReader, nil
}

func (m *mockOrchestrator) CreateNamespace(_ context.Context, _ string) error {
	return nil
}

func (m *mockOrchestrator) NamespaceExists(_ context.Context, _ string) (bool, error) {
	return true, nil
}

func setupWithMockOrch(orch orchestrator.Orchestrator) (*Handler, *store.MemoryStore) {
	s := store.NewMemory()
	h := New(slog.Default(), s, s, orch, email.NewMock(), 7*24*time.Hour, true, "registry.local", "test-secret", 1*time.Hour)
	return h, s
}

func TestStreamLogs(t *testing.T) {
	lines := "line one\nline two\nline three\n"
	orch := &mockOrchestrator{
		logsReader: io.NopCloser(strings.NewReader(lines)),
	}
	h, s := setupWithMockOrch(orch)

	svc, err := s.Create(context.Background(), models.DeployRequest{Name: "app", Image: "img"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/api/v1/services/{id}/logs", h.StreamLogs)

	req := httptest.NewRequest("GET", "/api/v1/services/"+svc.ID+"/logs?tail=3", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Fatalf("expected Content-Type text/event-stream, got %s", ct)
	}

	// SSE-Events parsen
	scanner := bufio.NewScanner(w.Body)
	var entries []models.LogEntry
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var entry models.LogEntry
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &entry); err != nil {
			t.Fatalf("failed to unmarshal log entry: %v", err)
		}
		entries = append(entries, entry)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 SSE events, got %d", len(entries))
	}
	if entries[0].Message != "line one" {
		t.Fatalf("expected message 'line one', got %q", entries[0].Message)
	}
}

func TestStreamLogsNotFound(t *testing.T) {
	orch := &mockOrchestrator{}
	h, _ := setupWithMockOrch(orch)

	r := chi.NewRouter()
	r.Get("/api/v1/services/{id}/logs", h.StreamLogs)

	req := httptest.NewRequest("GET", "/api/v1/services/nonexistent/logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestStreamLogsNoPods(t *testing.T) {
	orch := &mockOrchestrator{
		logsErr: orchestrator.ErrNoPods,
	}
	h, s := setupWithMockOrch(orch)

	svc, err := s.Create(context.Background(), models.DeployRequest{Name: "app", Image: "img"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/api/v1/services/{id}/logs", h.StreamLogs)

	req := httptest.NewRequest("GET", "/api/v1/services/"+svc.ID+"/logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestDetectStream(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "plain text defaults to stdout",
			line:     "connection established",
			expected: "stdout",
		},
		{
			name:     "JSON info level is stdout",
			line:     `{"level":"info","msg":"started"}`,
			expected: "stdout",
		},
		{
			name:     "JSON error level is stderr",
			line:     `{"level":"error","msg":"failed to connect"}`,
			expected: "stderr",
		},
		{
			name:     "JSON fatal level is stderr",
			line:     `{"level":"fatal","msg":"unrecoverable"}`,
			expected: "stderr",
		},
		{
			name:     "JSON severity key is recognized",
			line:     `{"severity":"error","message":"oops"}`,
			expected: "stderr",
		},
		{
			name:     "JSON warn level is stdout",
			line:     `{"level":"warn","msg":"deprecated"}`,
			expected: "stdout",
		},
		{
			name:     "JSON debug level is stdout",
			line:     `{"level":"debug","msg":"trace"}`,
			expected: "stdout",
		},
		{
			name:     "JSON critical is stderr",
			line:     `{"level":"critical","msg":"system down"}`,
			expected: "stderr",
		},
		{
			name:     "non-error JSON stays stdout",
			line:     `{"ts":12345,"msg":"ok"}`,
			expected: "stdout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectStream(tt.line)
			if result != tt.expected {
				t.Errorf("detectStream(%q) = %q, want %q", tt.line, result, tt.expected)
			}
		})
	}
}
