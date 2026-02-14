package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/max-cloud/shared/pkg/models"
)

func mockAPI() *httptest.Server {
	mux := http.NewServeMux()

	services := map[string]models.Service{
		"svc-1": {ID: "svc-1", Name: "app1", Image: "nginx:latest", Status: "ready", URL: "https://app1.maxcloud.dev"},
	}

	mux.HandleFunc("GET /api/v1/services", func(w http.ResponseWriter, r *http.Request) {
		var list []models.Service
		for _, s := range services {
			list = append(list, s)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(list)
	})

	mux.HandleFunc("POST /api/v1/services", func(w http.ResponseWriter, r *http.Request) {
		var req models.DeployRequest
		json.NewDecoder(r.Body).Decode(&req)
		svc := models.Service{
			ID:    "svc-new",
			Name:  req.Name,
			Image: req.Image,
			Status: "ready",
			URL:   "https://" + req.Name + ".maxcloud.dev",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(svc)
	})

	mux.HandleFunc("GET /api/v1/services/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		svc, ok := services[id]
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(svc)
	})

	mux.HandleFunc("DELETE /api/v1/services/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if _, ok := services[id]; !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	return httptest.NewServer(mux)
}

func TestClientListServices(t *testing.T) {
	srv := mockAPI()
	defer srv.Close()

	c := NewClient(srv.URL)
	services, err := c.ListServices()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}
	if services[0].Name != "app1" {
		t.Fatalf("expected name app1, got %s", services[0].Name)
	}
}

func TestClientDeploy(t *testing.T) {
	srv := mockAPI()
	defer srv.Close()

	c := NewClient(srv.URL)
	svc, err := c.Deploy(models.DeployRequest{Name: "newapp", Image: "alpine:latest"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.Name != "newapp" {
		t.Fatalf("expected name newapp, got %s", svc.Name)
	}
	if svc.ID != "svc-new" {
		t.Fatalf("expected ID svc-new, got %s", svc.ID)
	}
}

func TestClientGetService(t *testing.T) {
	srv := mockAPI()
	defer srv.Close()

	c := NewClient(srv.URL)
	svc, err := c.GetService("svc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.Name != "app1" {
		t.Fatalf("expected name app1, got %s", svc.Name)
	}
}

func TestClientGetServiceNotFound(t *testing.T) {
	srv := mockAPI()
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.GetService("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent service")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Fatalf("expected status 404, got %d", apiErr.StatusCode)
	}
}

func TestClientDeleteService(t *testing.T) {
	srv := mockAPI()
	defer srv.Close()

	c := NewClient(srv.URL)
	err := c.DeleteService("svc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientDeleteServiceNotFound(t *testing.T) {
	srv := mockAPI()
	defer srv.Close()

	c := NewClient(srv.URL)
	err := c.DeleteService("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent service")
	}
}
