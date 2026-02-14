package models

import "time"

// Service represents a deployed container service.
type Service struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	Status    ServiceStatus     `json:"status"`
	URL       string            `json:"url"`
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
	EnvVars map[string]string `json:"env_vars,omitempty"`
}

// LogEntry represents a single log line from a service.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Stream    string    `json:"stream"`
}
