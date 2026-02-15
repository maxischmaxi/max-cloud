package models

import "time"

// Organization repräsentiert einen Mandanten (Tenant) in max-cloud.
type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// User repräsentiert einen registrierten Benutzer.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// OrgRole definiert die Rolle eines Benutzers innerhalb einer Organisation.
type OrgRole string

const (
	OrgRoleAdmin  OrgRole = "admin"
	OrgRoleMember OrgRole = "member"
)

// APIKeyInfo enthält Metadaten zu einem API-Key (ohne den Schlüssel selbst).
type APIKeyInfo struct {
	ID         string     `json:"id"`
	Prefix     string     `json:"prefix"`
	Name       string     `json:"name"`
	OrgID      string     `json:"org_id"`
	UserID     string     `json:"user_id"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// RegisterRequest ist der Payload für die Registrierung.
type RegisterRequest struct {
	Email   string `json:"email"`
	OrgName string `json:"org_name"`
}

// RegisterResponse enthält die Antwort nach erfolgreicher Registrierung.
type RegisterResponse struct {
	User         User         `json:"user"`
	Organization Organization `json:"organization"`
	APIKey       string       `json:"api_key"`
}

// CreateAPIKeyRequest ist der Payload zum Erstellen eines neuen API-Keys.
type CreateAPIKeyRequest struct {
	Name string `json:"name"`
}

// CreateAPIKeyResponse enthält den neuen API-Key (einmalig sichtbar).
type CreateAPIKeyResponse struct {
	APIKey string     `json:"api_key"`
	Info   APIKeyInfo `json:"info"`
}

// AuthInfo enthält Informationen über den aktuell authentifizierten Benutzer.
type AuthInfo struct {
	User         User         `json:"user"`
	Organization Organization `json:"organization"`
	Role         OrgRole      `json:"role"`
}

// InviteStatus definiert den Status einer Einladung.
type InviteStatus string

const (
	InviteStatusPending  InviteStatus = "pending"
	InviteStatusAccepted InviteStatus = "accepted"
	InviteStatusExpired  InviteStatus = "expired"
	InviteStatusRevoked  InviteStatus = "revoked"
)

// Invitation repräsentiert eine Einladung zu einer Organisation.
type Invitation struct {
	ID        string       `json:"id"`
	OrgID     string       `json:"org_id"`
	OrgName   string       `json:"org_name"`
	Email     string       `json:"email"`
	Role      OrgRole      `json:"role"`
	Status    InviteStatus `json:"status"`
	InvitedBy string       `json:"invited_by"`
	ExpiresAt time.Time    `json:"expires_at"`
	CreatedAt time.Time    `json:"created_at"`
}

// InviteRequest ist der Payload zum Erstellen einer Einladung.
type InviteRequest struct {
	Email string  `json:"email"`
	Role  OrgRole `json:"role"`
}

// InviteResponse enthält die erstellte Einladung (Token nur im Dev-Modus).
type InviteResponse struct {
	Invitation Invitation `json:"invitation"`
	Token      string     `json:"token,omitempty"`
}

// AcceptInviteRequest ist der Payload zum Annehmen einer Einladung.
type AcceptInviteRequest struct {
	Token string `json:"token"`
}

// AcceptInviteResponse enthält das Ergebnis einer angenommenen Einladung.
type AcceptInviteResponse struct {
	User         User         `json:"user"`
	Organization Organization `json:"organization"`
	Role         OrgRole      `json:"role"`
	APIKey       string       `json:"api_key"`
}
