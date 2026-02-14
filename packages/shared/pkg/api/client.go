package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/max-cloud/shared/pkg/models"
)

// Client communicates with the max-cloud API server.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
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

// Deploy creates a new service.
func (c *Client) Deploy(req models.DeployRequest) (*models.Service, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.HTTPClient.Post(c.BaseURL+"/api/v1/services", "application/json", bytes.NewReader(body))
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
	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/v1/services")
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
	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/v1/services/" + id)
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
	req, err := http.NewRequest(http.MethodDelete, c.BaseURL+"/api/v1/services/"+id, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return parseAPIError(resp)
	}
	return nil
}

// APIError represents a structured error from the API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
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
