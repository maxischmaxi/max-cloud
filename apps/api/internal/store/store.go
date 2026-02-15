package store

import (
	"context"
	"errors"
	"time"

	"github.com/max-cloud/shared/pkg/models"
)

// ErrNotFound wird zurückgegeben, wenn ein Service nicht existiert.
var ErrNotFound = errors.New("service not found")

// ErrDuplicateService wird zurückgegeben, wenn ein Service mit diesem Namen bereits existiert.
var ErrDuplicateService = errors.New("service name already exists in this organization")

// ErrDuplicateEmail wird zurückgegeben, wenn die E-Mail bereits registriert ist.
var ErrDuplicateEmail = errors.New("email already registered")

// ErrDuplicateOrg wird zurückgegeben, wenn der Org-Name bereits existiert.
var ErrDuplicateOrg = errors.New("organization name already taken")

// ErrKeyNotFound wird zurückgegeben, wenn ein API-Key nicht existiert.
var ErrKeyNotFound = errors.New("api key not found")

// ErrInviteNotFound wird zurückgegeben, wenn eine Einladung nicht existiert.
var ErrInviteNotFound = errors.New("invite not found")

// ErrInviteExpired wird zurückgegeben, wenn eine Einladung abgelaufen ist.
var ErrInviteExpired = errors.New("invite expired")

// ErrAlreadyMember wird zurückgegeben, wenn der User bereits Mitglied der Org ist.
var ErrAlreadyMember = errors.New("user is already a member of this organization")

// ServiceStore definiert die Schnittstelle für Service-Persistenz.
type ServiceStore interface {
	Create(ctx context.Context, req models.DeployRequest) (models.Service, error)
	Get(ctx context.Context, id string) (models.Service, error)
	GetByName(ctx context.Context, name string) (models.Service, error)
	List(ctx context.Context) ([]models.Service, error)
	Delete(ctx context.Context, id string) error
	UpdateStatus(ctx context.Context, id string, status models.ServiceStatus, url string) error
}

// AuthStore definiert die Schnittstelle für Authentifizierung und Benutzerverwaltung.
type AuthStore interface {
	Register(ctx context.Context, email, orgName string) (models.User, models.Organization, string, error)
	ValidateAPIKey(ctx context.Context, rawKey string) (*models.APIKeyInfo, error)
	CreateAPIKey(ctx context.Context, orgID, userID, name string) (string, *models.APIKeyInfo, error)
	ListAPIKeys(ctx context.Context, orgID string) ([]models.APIKeyInfo, error)
	DeleteAPIKey(ctx context.Context, orgID, keyID string) error
	GetAuthInfo(ctx context.Context, orgID, userID string) (*models.AuthInfo, error)
	UpdateAPIKeyLastUsed(ctx context.Context, keyID string) error
	CreateInvite(ctx context.Context, orgID, email string, role models.OrgRole, invitedBy string, expiresAt time.Time) (models.Invitation, string, error)
	AcceptInvite(ctx context.Context, rawToken string) (models.User, models.Organization, models.OrgRole, string, error)
	ListInvites(ctx context.Context, orgID string) ([]models.Invitation, error)
	RevokeInvite(ctx context.Context, orgID, inviteID string) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	EnsureDevOrg(ctx context.Context, devOrgID string) error
}
