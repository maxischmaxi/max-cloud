package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
			ID:     "svc-new",
			Name:   req.Name,
			Image:  req.Image,
			Status: "ready",
			URL:    "https://" + req.Name + ".maxcloud.dev",
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

	mux.HandleFunc("GET /api/v1/services/{id}/logs", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if _, ok := services[id]; !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}
		for i := 0; i < 2; i++ {
			entry := models.LogEntry{
				Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				Message:   fmt.Sprintf("log line %d", i+1),
				Stream:    "stdout",
			}
			data, _ := json.Marshal(entry)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	})

	// Auth-Endpoints
	mux.HandleFunc("POST /api/v1/auth/register", func(w http.ResponseWriter, r *http.Request) {
		var req models.RegisterRequest
		json.NewDecoder(r.Body).Decode(&req)
		resp := models.RegisterResponse{
			User:         models.User{ID: "user-1", Email: req.Email, CreatedAt: time.Now()},
			Organization: models.Organization{ID: "org-1", Name: req.OrgName, CreatedAt: time.Now()},
			APIKey:       "mc_abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("POST /api/v1/auth/api-keys", func(w http.ResponseWriter, r *http.Request) {
		var req models.CreateAPIKeyRequest
		json.NewDecoder(r.Body).Decode(&req)
		resp := models.CreateAPIKeyResponse{
			APIKey: "mc_newkey1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			Info: models.APIKeyInfo{
				ID:        "key-new",
				Prefix:    "newkey12",
				Name:      req.Name,
				OrgID:     "org-1",
				UserID:    "user-1",
				CreatedAt: time.Now(),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("GET /api/v1/auth/api-keys", func(w http.ResponseWriter, r *http.Request) {
		keys := []models.APIKeyInfo{
			{ID: "key-1", Prefix: "abcdef12", Name: "default", OrgID: "org-1", UserID: "user-1", CreatedAt: time.Now()},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(keys)
	})

	mux.HandleFunc("DELETE /api/v1/auth/api-keys/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /api/v1/auth/status", func(w http.ResponseWriter, r *http.Request) {
		info := models.AuthInfo{
			User:         models.User{ID: "user-1", Email: "test@example.com", CreatedAt: time.Now()},
			Organization: models.Organization{ID: "org-1", Name: "TestOrg", CreatedAt: time.Now()},
			Role:         models.OrgRoleAdmin,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	})

	// Invite-Endpoints
	mux.HandleFunc("POST /api/v1/auth/invites", func(w http.ResponseWriter, r *http.Request) {
		var req models.InviteRequest
		json.NewDecoder(r.Body).Decode(&req)
		resp := models.InviteResponse{
			Invitation: models.Invitation{
				ID:      "inv-1",
				OrgID:   "org-1",
				OrgName: "TestOrg",
				Email:   req.Email,
				Role:    req.Role,
				Status:  models.InviteStatusPending,
			},
			Token: "mci_devtoken1234567890abcdef1234567890abcdef1234567890abcdef12345678",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("GET /api/v1/auth/invites", func(w http.ResponseWriter, r *http.Request) {
		invites := []models.Invitation{
			{ID: "inv-1", OrgID: "org-1", OrgName: "TestOrg", Email: "a@example.com", Role: models.OrgRoleMember, Status: models.InviteStatusPending},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(invites)
	})

	mux.HandleFunc("DELETE /api/v1/auth/invites/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("POST /api/v1/auth/accept-invite", func(w http.ResponseWriter, r *http.Request) {
		resp := models.AcceptInviteResponse{
			User:         models.User{ID: "user-2", Email: "new@example.com", CreatedAt: time.Now()},
			Organization: models.Organization{ID: "org-1", Name: "TestOrg", CreatedAt: time.Now()},
			Role:         models.OrgRoleMember,
			APIKey:       "mc_newuserkey1234567890abcdef1234567890abcdef1234567890abcdef123456",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
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

func TestClientStreamLogs(t *testing.T) {
	srv := mockAPI()
	defer srv.Close()

	c := NewClient(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ls, err := c.StreamLogs(ctx, "svc-1", false, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer ls.Close()

	var entries []models.LogEntry
	for entry := range ls.Events {
		entries = append(entries, entry)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 events, got %d", len(entries))
	}
	if entries[0].Message != "log line 1" {
		t.Fatalf("expected 'log line 1', got %q", entries[0].Message)
	}
	if entries[1].Message != "log line 2" {
		t.Fatalf("expected 'log line 2', got %q", entries[1].Message)
	}
}

func TestClientStreamLogsNotFound(t *testing.T) {
	srv := mockAPI()
	defer srv.Close()

	c := NewClient(srv.URL)
	ctx := context.Background()

	_, err := c.StreamLogs(ctx, "nonexistent", false, 10)
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

func TestClientRegister(t *testing.T) {
	srv := mockAPI()
	defer srv.Close()

	c := NewClient(srv.URL)
	resp, err := c.Register(models.RegisterRequest{Email: "test@example.com", OrgName: "TestOrg"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.User.Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %s", resp.User.Email)
	}
	if resp.APIKey == "" {
		t.Fatal("expected non-empty API key")
	}
}

func TestClientAuthHeader(t *testing.T) {
	var receivedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]models.Service{})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	c.Token = "mc_testkey123"

	_, err := c.ListServices()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedAuth != "Bearer mc_testkey123" {
		t.Fatalf("expected Authorization header 'Bearer mc_testkey123', got %q", receivedAuth)
	}
}

func TestClientAuthStatus(t *testing.T) {
	srv := mockAPI()
	defer srv.Close()

	c := NewClient(srv.URL)
	c.Token = "mc_testkey"

	info, err := c.AuthStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.User.Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %s", info.User.Email)
	}
	if info.Role != models.OrgRoleAdmin {
		t.Fatalf("expected role admin, got %s", info.Role)
	}
}

func TestClientCreateInvite(t *testing.T) {
	srv := mockAPI()
	defer srv.Close()

	c := NewClient(srv.URL)
	c.Token = "mc_testkey"

	resp, err := c.CreateInvite(models.InviteRequest{Email: "new@example.com", Role: models.OrgRoleMember})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Invitation.Email != "new@example.com" {
		t.Fatalf("expected email new@example.com, got %s", resp.Invitation.Email)
	}
	if resp.Token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestClientListInvites(t *testing.T) {
	srv := mockAPI()
	defer srv.Close()

	c := NewClient(srv.URL)
	c.Token = "mc_testkey"

	invites, err := c.ListInvites()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(invites) != 1 {
		t.Fatalf("expected 1 invite, got %d", len(invites))
	}
	if invites[0].Email != "a@example.com" {
		t.Fatalf("expected email a@example.com, got %s", invites[0].Email)
	}
}

func TestClientRevokeInvite(t *testing.T) {
	srv := mockAPI()
	defer srv.Close()

	c := NewClient(srv.URL)
	c.Token = "mc_testkey"

	err := c.RevokeInvite("inv-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientAcceptInvite(t *testing.T) {
	srv := mockAPI()
	defer srv.Close()

	c := NewClient(srv.URL)

	resp, err := c.AcceptInvite(models.AcceptInviteRequest{Token: "mci_testtoken"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.User.Email != "new@example.com" {
		t.Fatalf("expected email new@example.com, got %s", resp.User.Email)
	}
	if resp.APIKey == "" {
		t.Fatal("expected non-empty API key")
	}
	if resp.Organization.Name != "TestOrg" {
		t.Fatalf("expected org TestOrg, got %s", resp.Organization.Name)
	}
}
