package models

import "time"

// Service represents a deployed container service.
type Service struct {
	ID        string            `json:"id"`
	OrgID     string            `json:"org_id,omitempty"`
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	Status    ServiceStatus     `json:"status"`
	URL       string            `json:"url"`
	Port      int               `json:"port,omitempty"`
	Command   []string          `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	EnvVars   map[string]string `json:"env_vars,omitempty"`
	MinScale  int               `json:"min_scale"`
	MaxScale  int               `json:"max_scale"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// ServiceStatus represents the current state of a service.
type ServiceStatus string

const (
	ServiceStatusReady    ServiceStatus = "ready"
	ServiceStatusPending  ServiceStatus = "pending"
	ServiceStatusFailed   ServiceStatus = "failed"
	ServiceStatusDeleting ServiceStatus = "deleting"
)

// Revision represents an immutable snapshot of a service configuration.
type Revision struct {
	ID        string    `json:"id"`
	ServiceID string    `json:"service_id"`
	Image     string    `json:"image"`
	Traffic   int       `json:"traffic"`
	CreatedAt time.Time `json:"created_at"`
}

// DeployRequest is the payload for deploying a new service.
type DeployRequest struct {
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	Port    int               `json:"port,omitempty"`
	Command []string          `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	EnvVars map[string]string `json:"env_vars,omitempty"`
}

// LogEntry represents a single log line from a service.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Stream    string    `json:"stream"`
}

// RegistryTokenRequest für Token-Anfrage an die Registry.
type RegistryTokenRequest struct {
	Scope string `json:"scope,omitempty"`
}

// RegistryTokenResponse für Token-Antwort (Docker Registry Token Format).
type RegistryTokenResponse struct {
	Token        string `json:"token"`
	AccessToken  string `json:"access_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
	IssuedAt     string `json:"issued_at"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// Image represents an image in the registry.
type Image struct {
	Name      string    `json:"name"`
	Tags      []string  `json:"tags"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

// Repository represents a repository in the registry.
type Repository struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Tags      []string `json:"tags,omitempty"`
}
