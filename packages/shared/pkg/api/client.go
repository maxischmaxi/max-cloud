package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/max-cloud/shared/pkg/models"
)

// Client communicates with the max-cloud API server.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Token      string
}

// NewClient creates a new API client.
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest erstellt und führt einen HTTP-Request mit Auth-Header aus.
func (c *Client) doRequest(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	return c.HTTPClient.Do(req)
}

// Deploy creates a new service.
func (c *Client) Deploy(req models.DeployRequest) (*models.Service, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest(http.MethodPost, c.BaseURL+"/api/v1/services", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, parseAPIError(resp)
	}

	var svc models.Service
	if err := json.NewDecoder(resp.Body).Decode(&svc); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &svc, nil
}

// ListServices returns all services.
func (c *Client) ListServices() ([]models.Service, error) {
	resp, err := c.doRequest(http.MethodGet, c.BaseURL+"/api/v1/services", nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var services []models.Service
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return services, nil
}

// GetService returns a single service by ID.
func (c *Client) GetService(id string) (*models.Service, error) {
	resp, err := c.doRequest(http.MethodGet, c.BaseURL+"/api/v1/services/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var svc models.Service
	if err := json.NewDecoder(resp.Body).Decode(&svc); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &svc, nil
}

// DeleteService deletes a service by ID.
func (c *Client) DeleteService(id string) error {
	resp, err := c.doRequest(http.MethodDelete, c.BaseURL+"/api/v1/services/"+id, nil)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return parseAPIError(resp)
	}
	return nil
}

// Register erstellt einen neuen Account.
func (c *Client) Register(req models.RegisterRequest) (*models.RegisterResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest(http.MethodPost, c.BaseURL+"/api/v1/auth/register", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, parseAPIError(resp)
	}

	var result models.RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// CreateAPIKey erstellt einen neuen API-Key.
func (c *Client) CreateAPIKey(req models.CreateAPIKeyRequest) (*models.CreateAPIKeyResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest(http.MethodPost, c.BaseURL+"/api/v1/auth/api-keys", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, parseAPIError(resp)
	}

	var result models.CreateAPIKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// ListAPIKeys gibt alle API-Keys zurück.
func (c *Client) ListAPIKeys() ([]models.APIKeyInfo, error) {
	resp, err := c.doRequest(http.MethodGet, c.BaseURL+"/api/v1/auth/api-keys", nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var keys []models.APIKeyInfo
	if err := json.NewDecoder(resp.Body).Decode(&keys); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return keys, nil
}

// DeleteAPIKey löscht einen API-Key.
func (c *Client) DeleteAPIKey(id string) error {
	resp, err := c.doRequest(http.MethodDelete, c.BaseURL+"/api/v1/auth/api-keys/"+id, nil)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return parseAPIError(resp)
	}
	return nil
}

// AuthStatus gibt Informationen über den aktuellen Benutzer zurück.
func (c *Client) AuthStatus() (*models.AuthInfo, error) {
	resp, err := c.doRequest(http.MethodGet, c.BaseURL+"/api/v1/auth/status", nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var info models.AuthInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &info, nil
}

// APIError represents a structured error from the API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

// LogStream repräsentiert einen laufenden Log-Stream.
type LogStream struct {
	Events <-chan models.LogEntry
	cancel context.CancelFunc
	resp   *http.Response
}

// Close beendet den Log-Stream.
func (ls *LogStream) Close() {
	ls.cancel()
	if ls.resp != nil {
		ls.resp.Body.Close()
	}
}

// StreamLogs öffnet einen SSE-Stream für Container-Logs.
func (c *Client) StreamLogs(ctx context.Context, id string, follow bool, tail int) (*LogStream, error) {
	ctx, cancel := context.WithCancel(ctx)

	url := fmt.Sprintf("%s/api/v1/services/%s/logs?follow=%t&tail=%d", c.BaseURL, id, follow, tail)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create request: %w", err)
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	// Eigener Client ohne Timeout für langlebiges SSE-Streaming
	sseClient := &http.Client{}
	resp, err := sseClient.Do(req)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		cancel()
		return nil, parseAPIError(resp)
	}

	ch := make(chan models.LogEntry, 64)
	ls := &LogStream{
		Events: ch,
		cancel: cancel,
		resp:   resp,
	}

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var entry models.LogEntry
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &entry); err != nil {
				continue
			}
			select {
			case ch <- entry:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ls, nil
}

// CreateInvite erstellt eine neue Einladung.
func (c *Client) CreateInvite(req models.InviteRequest) (*models.InviteResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest(http.MethodPost, c.BaseURL+"/api/v1/auth/invites", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, parseAPIError(resp)
	}

	var result models.InviteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// ListInvites gibt alle pending Einladungen zurück.
func (c *Client) ListInvites() ([]models.Invitation, error) {
	resp, err := c.doRequest(http.MethodGet, c.BaseURL+"/api/v1/auth/invites", nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var invites []models.Invitation
	if err := json.NewDecoder(resp.Body).Decode(&invites); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return invites, nil
}

// RevokeInvite widerruft eine Einladung.
func (c *Client) RevokeInvite(id string) error {
	resp, err := c.doRequest(http.MethodDelete, c.BaseURL+"/api/v1/auth/invites/"+id, nil)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return parseAPIError(resp)
	}
	return nil
}

// AcceptInvite nimmt eine Einladung an.
func (c *Client) AcceptInvite(req models.AcceptInviteRequest) (*models.AcceptInviteResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest(http.MethodPost, c.BaseURL+"/api/v1/auth/accept-invite", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, parseAPIError(resp)
	}

	var result models.AcceptInviteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

func parseAPIError(resp *http.Response) error {
	var errBody struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&errBody); err != nil {
		return &APIError{StatusCode: resp.StatusCode, Message: resp.Status}
	}
	msg := errBody.Error
	if msg == "" {
		msg = resp.Status
	}
	return &APIError{StatusCode: resp.StatusCode, Message: msg}
}

// GetRegistryToken holt ein JWT für die Docker Registry.
func (c *Client) GetRegistryToken(scope string) (*models.RegistryTokenResponse, error) {
	url := c.BaseURL + "/api/v1/registry/token"
	if scope != "" {
		url = url + "?scope=" + scope
	}

	resp, err := c.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result models.RegistryTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}
