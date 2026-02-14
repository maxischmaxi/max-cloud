package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/max-cloud/api/internal/store"
	"github.com/max-cloud/shared/pkg/models"
)

func setup() (*Handler, *store.Store) {
	s := store.New()
	h := New(slog.Default(), s)
	return h, s
}

// chiContext wraps a handler call with chi URL params.
func withChiParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(r.Context())
}

func TestHealth(t *testing.T) {
	h, _ := setup()
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %s", body["status"])
	}
}

func TestCreateService(t *testing.T) {
	h, _ := setup()
	payload := `{"name":"myapp","image":"nginx:latest"}`
	req := httptest.NewRequest("POST", "/api/v1/services", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()

	h.CreateService(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var svc models.Service
	json.NewDecoder(w.Body).Decode(&svc)
	if svc.Name != "myapp" {
		t.Fatalf("expected name myapp, got %s", svc.Name)
	}
	if svc.Image != "nginx:latest" {
		t.Fatalf("expected image nginx:latest, got %s", svc.Image)
	}
	if svc.ID == "" {
		t.Fatal("expected non-empty ID")
	}
}

func TestCreateServiceValidation(t *testing.T) {
	h, _ := setup()

	tests := []struct {
		name    string
		payload string
	}{
		{"missing name", `{"image":"nginx:latest"}`},
		{"missing image", `{"name":"myapp"}`},
		{"invalid json", `{invalid`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/services", bytes.NewBufferString(tt.payload))
			w := httptest.NewRecorder()
			h.CreateService(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", w.Code)
			}
		})
	}
}

func TestListServicesEmpty(t *testing.T) {
	h, _ := setup()
	req := httptest.NewRequest("GET", "/api/v1/services", nil)
	w := httptest.NewRecorder()

	h.ListServices(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var services []models.Service
	json.NewDecoder(w.Body).Decode(&services)
	if len(services) != 0 {
		t.Fatalf("expected 0 services, got %d", len(services))
	}
}

func TestListServicesWithData(t *testing.T) {
	h, s := setup()
	s.Create(models.DeployRequest{Name: "a", Image: "img"})
	s.Create(models.DeployRequest{Name: "b", Image: "img"})

	req := httptest.NewRequest("GET", "/api/v1/services", nil)
	w := httptest.NewRecorder()

	h.ListServices(w, req)

	var services []models.Service
	json.NewDecoder(w.Body).Decode(&services)
	if len(services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(services))
	}
}

func TestGetServiceAndDelete(t *testing.T) {
	h, s := setup()
	created := s.Create(models.DeployRequest{Name: "app", Image: "img"})

	// Use chi router for proper URL param extraction
	r := chi.NewRouter()
	r.Get("/api/v1/services/{id}", h.GetService)
	r.Delete("/api/v1/services/{id}", h.DeleteService)

	// Test Get
	req := httptest.NewRequest("GET", "/api/v1/services/"+created.ID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var svc models.Service
	json.NewDecoder(w.Body).Decode(&svc)
	if svc.ID != created.ID {
		t.Fatalf("expected ID %s, got %s", created.ID, svc.ID)
	}

	// Test Delete
	req = httptest.NewRequest("DELETE", "/api/v1/services/"+created.ID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	// Test Get after delete -> 404
	req = httptest.NewRequest("GET", "/api/v1/services/"+created.ID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetServiceNotFound(t *testing.T) {
	h, _ := setup()

	r := chi.NewRouter()
	r.Get("/api/v1/services/{id}", h.GetService)

	req := httptest.NewRequest("GET", "/api/v1/services/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeleteServiceNotFound(t *testing.T) {
	h, _ := setup()

	r := chi.NewRouter()
	r.Delete("/api/v1/services/{id}", h.DeleteService)

	req := httptest.NewRequest("DELETE", "/api/v1/services/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
