package store

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/max-cloud/shared/pkg/models"
)

// Store provides thread-safe in-memory storage for services.
type Store struct {
	mu       sync.RWMutex
	services map[string]models.Service
}

// New creates a new Store.
func New() *Store {
	return &Store{
		services: make(map[string]models.Service),
	}
}

// Create adds a new service to the store.
func (s *Store) Create(req models.DeployRequest) models.Service {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	svc := models.Service{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Image:     req.Image,
		Status:    models.ServiceStatusReady,
		URL:       fmt.Sprintf("https://%s.maxcloud.dev", req.Name),
		EnvVars:   req.EnvVars,
		MinScale:  0,
		MaxScale:  10,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.services[svc.ID] = svc
	return svc
}

// Get returns a service by ID, or false if not found.
func (s *Store) Get(id string) (models.Service, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	svc, ok := s.services[id]
	return svc, ok
}

// List returns all services.
func (s *Store) List() []models.Service {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]models.Service, 0, len(s.services))
	for _, svc := range s.services {
		result = append(result, svc)
	}
	return result
}

// Delete removes a service by ID. Returns false if not found.
func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.services[id]; !ok {
		return false
	}
	delete(s.services, id)
	return true
}
