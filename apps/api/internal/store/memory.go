package store

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/max-cloud/api/internal/auth"
	"github.com/max-cloud/shared/pkg/models"
)

// MemoryStore provides thread-safe in-memory storage for services.
type MemoryStore struct {
	mu       sync.RWMutex
	services map[string]models.Service

	// Auth-Daten
	orgs         map[string]models.Organization       // id → org
	users        map[string]models.User               // id → user
	orgMembers   map[string]map[string]models.OrgRole // orgID → userID → role
	apiKeys      map[string][]apiKeyEntry             // prefix → entries
	apiKeysByID  map[string]*apiKeyEntry              // keyID → entry
	emailIndex   map[string]string                    // email → userID
	orgNameIndex map[string]bool                      // orgName → exists

	// Invite-Daten
	invitations  map[string]models.Invitation  // inviteID → invitation
	inviteTokens map[string][]inviteTokenEntry // tokenPrefix → entries
}

type inviteTokenEntry struct {
	inviteID string
	hash     string
}

type apiKeyEntry struct {
	info models.APIKeyInfo
	hash string
}

// NewMemory creates a new MemoryStore.
func NewMemory() *MemoryStore {
	return &MemoryStore{
		services:     make(map[string]models.Service),
		orgs:         make(map[string]models.Organization),
		users:        make(map[string]models.User),
		orgMembers:   make(map[string]map[string]models.OrgRole),
		apiKeys:      make(map[string][]apiKeyEntry),
		apiKeysByID:  make(map[string]*apiKeyEntry),
		emailIndex:   make(map[string]string),
		orgNameIndex: make(map[string]bool),
		invitations:  make(map[string]models.Invitation),
		inviteTokens: make(map[string][]inviteTokenEntry),
	}
}

// Create adds a new service to the store.
func (s *MemoryStore) Create(ctx context.Context, req models.DeployRequest) (models.Service, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	orgID, hasOrgID := auth.OrgIDFromContext(ctx)

	for _, svc := range s.services {
		if svc.Name == req.Name {
			if hasOrgID && svc.OrgID == orgID {
				return models.Service{}, ErrDuplicateService
			}
			if !hasOrgID && svc.OrgID == "" {
				return models.Service{}, ErrDuplicateService
			}
		}
	}

	now := time.Now()
	svc := models.Service{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Image:     req.Image,
		Status:    models.ServiceStatusPending,
		Port:      req.Port,
		Command:   req.Command,
		Args:      req.Args,
		EnvVars:   req.EnvVars,
		MinScale:  0,
		MaxScale:  10,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if hasOrgID {
		svc.OrgID = orgID
	}

	s.services[svc.ID] = svc
	return svc, nil
}

// Get returns a service by ID. Returns ErrNotFound if not found.
func (s *MemoryStore) Get(ctx context.Context, id string) (models.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	svc, ok := s.services[id]
	if !ok {
		return models.Service{}, ErrNotFound
	}

	if orgID, ok := auth.OrgIDFromContext(ctx); ok {
		if svc.OrgID != orgID {
			return models.Service{}, ErrNotFound
		}
	}

	return svc, nil
}

// GetByName returns a service by name.
func (s *MemoryStore) GetByName(ctx context.Context, name string) (models.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, svc := range s.services {
		if svc.Name == name {
			if orgID, ok := auth.OrgIDFromContext(ctx); ok {
				if svc.OrgID != orgID {
					return models.Service{}, ErrNotFound
				}
			}
			return svc, nil
		}
	}

	return models.Service{}, ErrNotFound
}

// List returns all services.
func (s *MemoryStore) List(ctx context.Context) ([]models.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	orgID, hasTenant := auth.OrgIDFromContext(ctx)

	result := make([]models.Service, 0, len(s.services))
	for _, svc := range s.services {
		if hasTenant && svc.OrgID != orgID {
			continue
		}
		result = append(result, svc)
	}
	return result, nil
}

// Delete removes a service by ID. Returns ErrNotFound if not found.
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	svc, ok := s.services[id]
	if !ok {
		return ErrNotFound
	}

	if orgID, ok := auth.OrgIDFromContext(ctx); ok {
		if svc.OrgID != orgID {
			return ErrNotFound
		}
	}

	delete(s.services, id)
	return nil
}

// UpdateStatus setzt Status und URL eines Services.
func (s *MemoryStore) UpdateStatus(_ context.Context, id string, status models.ServiceStatus, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	svc, ok := s.services[id]
	if !ok {
		return ErrNotFound
	}
	svc.Status = status
	if url != "" {
		svc.URL = url
	}
	svc.UpdatedAt = time.Now()
	s.services[id] = svc
	return nil
}
