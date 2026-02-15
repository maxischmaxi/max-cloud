package store

import (
	"context"
	"crypto/subtle"
	"time"

	"github.com/google/uuid"
	"github.com/max-cloud/shared/pkg/models"
)

// Register erstellt einen neuen User, eine neue Organisation und einen initialen API-Key.
func (s *MemoryStore) Register(_ context.Context, email, orgName string) (models.User, models.Organization, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.emailIndex[email]; exists {
		return models.User{}, models.Organization{}, "", ErrDuplicateEmail
	}
	if s.orgNameIndex[orgName] {
		return models.User{}, models.Organization{}, "", ErrDuplicateOrg
	}

	now := time.Now()

	org := models.Organization{
		ID:        uuid.New().String(),
		Name:      orgName,
		CreatedAt: now,
	}

	user := models.User{
		ID:        uuid.New().String(),
		Email:     email,
		CreatedAt: now,
	}

	rawKey, keyHash, prefix, err := generateAPIKey()
	if err != nil {
		return models.User{}, models.Organization{}, "", err
	}

	keyInfo := models.APIKeyInfo{
		ID:        uuid.New().String(),
		Prefix:    prefix,
		Name:      "default",
		OrgID:     org.ID,
		UserID:    user.ID,
		CreatedAt: now,
	}

	entry := apiKeyEntry{info: keyInfo, hash: keyHash}

	// Alles speichern
	s.orgs[org.ID] = org
	s.users[user.ID] = user
	s.orgMembers[org.ID] = map[string]models.OrgRole{user.ID: models.OrgRoleAdmin}
	s.apiKeys[prefix] = append(s.apiKeys[prefix], entry)
	s.apiKeysByID[keyInfo.ID] = &s.apiKeys[prefix][len(s.apiKeys[prefix])-1]
	s.emailIndex[email] = user.ID
	s.orgNameIndex[orgName] = true

	return user, org, rawKey, nil
}

// ValidateAPIKey prüft einen API-Key und gibt die zugehörigen Informationen zurück.
func (s *MemoryStore) ValidateAPIKey(_ context.Context, rawKey string) (*models.APIKeyInfo, error) {
	prefix, err := extractPrefix(rawKey)
	if err != nil {
		return nil, ErrKeyNotFound
	}

	hash := hashAPIKey(rawKey)

	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, ok := s.apiKeys[prefix]
	if !ok {
		return nil, ErrKeyNotFound
	}

	for _, entry := range entries {
		if subtle.ConstantTimeCompare([]byte(entry.hash), []byte(hash)) == 1 {
			if entry.info.ExpiresAt != nil && entry.info.ExpiresAt.Before(time.Now()) {
				return nil, ErrKeyNotFound
			}
			info := entry.info
			return &info, nil
		}
	}

	return nil, ErrKeyNotFound
}

// CreateAPIKey erstellt einen neuen API-Key für eine Organisation/User.
func (s *MemoryStore) CreateAPIKey(_ context.Context, orgID, userID, name string) (string, *models.APIKeyInfo, error) {
	rawKey, keyHash, prefix, err := generateAPIKey()
	if err != nil {
		return "", nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	keyInfo := models.APIKeyInfo{
		ID:        uuid.New().String(),
		Prefix:    prefix,
		Name:      name,
		OrgID:     orgID,
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	entry := apiKeyEntry{info: keyInfo, hash: keyHash}
	s.apiKeys[prefix] = append(s.apiKeys[prefix], entry)
	s.apiKeysByID[keyInfo.ID] = &s.apiKeys[prefix][len(s.apiKeys[prefix])-1]

	return rawKey, &keyInfo, nil
}

// ListAPIKeys gibt alle API-Keys einer Organisation zurück.
func (s *MemoryStore) ListAPIKeys(_ context.Context, orgID string) ([]models.APIKeyInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []models.APIKeyInfo
	for _, entries := range s.apiKeys {
		for _, entry := range entries {
			if entry.info.OrgID == orgID {
				result = append(result, entry.info)
			}
		}
	}
	if result == nil {
		result = []models.APIKeyInfo{}
	}
	return result, nil
}

// DeleteAPIKey löscht einen API-Key anhand seiner ID.
func (s *MemoryStore) DeleteAPIKey(_ context.Context, orgID, keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.apiKeysByID[keyID]
	if !ok || entry.info.OrgID != orgID {
		return ErrKeyNotFound
	}

	prefix := entry.info.Prefix
	entries := s.apiKeys[prefix]
	for i, e := range entries {
		if e.info.ID == keyID {
			s.apiKeys[prefix] = append(entries[:i], entries[i+1:]...)
			break
		}
	}
	delete(s.apiKeysByID, keyID)

	return nil
}

// GetAuthInfo gibt Informationen über den authentifizierten Benutzer zurück.
func (s *MemoryStore) GetAuthInfo(_ context.Context, orgID, userID string) (*models.AuthInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	org, ok := s.orgs[orgID]
	if !ok {
		return nil, ErrNotFound
	}

	user, ok := s.users[userID]
	if !ok {
		return nil, ErrNotFound
	}

	members, ok := s.orgMembers[orgID]
	if !ok {
		return nil, ErrNotFound
	}

	role, ok := members[userID]
	if !ok {
		return nil, ErrNotFound
	}

	return &models.AuthInfo{
		User:         user,
		Organization: org,
		Role:         role,
	}, nil
}

// UpdateAPIKeyLastUsed aktualisiert den Zeitstempel der letzten Nutzung eines API-Keys.
func (s *MemoryStore) UpdateAPIKeyLastUsed(_ context.Context, keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.apiKeysByID[keyID]
	if !ok {
		return ErrKeyNotFound
	}

	now := time.Now()
	entry.info.LastUsedAt = &now
	return nil
}

// CreateInvite erstellt eine neue Einladung für eine Organisation.
func (s *MemoryStore) CreateInvite(_ context.Context, orgID, email string, role models.OrgRole, invitedBy string, expiresAt time.Time) (models.Invitation, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Prüfen ob User bereits Mitglied ist
	if userID, exists := s.emailIndex[email]; exists {
		if members, ok := s.orgMembers[orgID]; ok {
			if _, isMember := members[userID]; isMember {
				return models.Invitation{}, "", ErrAlreadyMember
			}
		}
	}

	rawToken, tokenHash, tokenPrefix, err := generateInviteToken()
	if err != nil {
		return models.Invitation{}, "", err
	}

	org, ok := s.orgs[orgID]
	if !ok {
		return models.Invitation{}, "", ErrNotFound
	}

	now := time.Now()
	invite := models.Invitation{
		ID:        uuid.New().String(),
		OrgID:     orgID,
		OrgName:   org.Name,
		Email:     email,
		Role:      role,
		Status:    models.InviteStatusPending,
		InvitedBy: invitedBy,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}

	s.invitations[invite.ID] = invite
	s.inviteTokens[tokenPrefix] = append(s.inviteTokens[tokenPrefix], inviteTokenEntry{
		inviteID: invite.ID,
		hash:     tokenHash,
	})

	return invite, rawToken, nil
}

// AcceptInvite nimmt eine Einladung an, erstellt ggf. den User und fügt ihn zur Org hinzu.
func (s *MemoryStore) AcceptInvite(_ context.Context, rawToken string) (models.User, models.Organization, models.OrgRole, string, error) {
	prefix, err := extractInvitePrefix(rawToken)
	if err != nil {
		return models.User{}, models.Organization{}, "", "", ErrInviteNotFound
	}

	tokenHash := hashInviteToken(rawToken)

	s.mu.Lock()
	defer s.mu.Unlock()

	entries, ok := s.inviteTokens[prefix]
	if !ok {
		return models.User{}, models.Organization{}, "", "", ErrInviteNotFound
	}

	var invite models.Invitation
	var found bool
	for _, entry := range entries {
		if subtle.ConstantTimeCompare([]byte(entry.hash), []byte(tokenHash)) == 1 {
			inv, exists := s.invitations[entry.inviteID]
			if !exists {
				return models.User{}, models.Organization{}, "", "", ErrInviteNotFound
			}
			invite = inv
			found = true
			break
		}
	}

	if !found {
		return models.User{}, models.Organization{}, "", "", ErrInviteNotFound
	}

	if invite.Status != models.InviteStatusPending {
		return models.User{}, models.Organization{}, "", "", ErrInviteNotFound
	}

	if time.Now().After(invite.ExpiresAt) {
		invite.Status = models.InviteStatusExpired
		s.invitations[invite.ID] = invite
		return models.User{}, models.Organization{}, "", "", ErrInviteExpired
	}

	// User finden oder erstellen
	var user models.User
	if userID, exists := s.emailIndex[invite.Email]; exists {
		user = s.users[userID]
	} else {
		now := time.Now()
		user = models.User{
			ID:        uuid.New().String(),
			Email:     invite.Email,
			CreatedAt: now,
		}
		s.users[user.ID] = user
		s.emailIndex[invite.Email] = user.ID
	}

	// Zur Org hinzufügen
	if s.orgMembers[invite.OrgID] == nil {
		s.orgMembers[invite.OrgID] = make(map[string]models.OrgRole)
	}
	s.orgMembers[invite.OrgID][user.ID] = invite.Role

	// API-Key generieren
	rawKey, keyHash, keyPrefix, err := generateAPIKey()
	if err != nil {
		return models.User{}, models.Organization{}, "", "", err
	}

	keyInfo := models.APIKeyInfo{
		ID:        uuid.New().String(),
		Prefix:    keyPrefix,
		Name:      "default",
		OrgID:     invite.OrgID,
		UserID:    user.ID,
		CreatedAt: time.Now(),
	}
	entry := apiKeyEntry{info: keyInfo, hash: keyHash}
	s.apiKeys[keyPrefix] = append(s.apiKeys[keyPrefix], entry)
	s.apiKeysByID[keyInfo.ID] = &s.apiKeys[keyPrefix][len(s.apiKeys[keyPrefix])-1]

	// Invite als accepted markieren
	invite.Status = models.InviteStatusAccepted
	s.invitations[invite.ID] = invite

	org := s.orgs[invite.OrgID]

	return user, org, invite.Role, rawKey, nil
}

// ListInvites gibt alle pending Einladungen einer Organisation zurück.
func (s *MemoryStore) ListInvites(_ context.Context, orgID string) ([]models.Invitation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []models.Invitation
	for _, inv := range s.invitations {
		if inv.OrgID == orgID && inv.Status == models.InviteStatusPending {
			result = append(result, inv)
		}
	}
	if result == nil {
		result = []models.Invitation{}
	}
	return result, nil
}

// RevokeInvite widerruft eine Einladung.
func (s *MemoryStore) RevokeInvite(_ context.Context, orgID, inviteID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	inv, ok := s.invitations[inviteID]
	if !ok || inv.OrgID != orgID {
		return ErrInviteNotFound
	}
	if inv.Status != models.InviteStatusPending {
		return ErrInviteNotFound
	}

	inv.Status = models.InviteStatusRevoked
	s.invitations[inviteID] = inv
	return nil
}

// GetUserByEmail sucht einen User anhand seiner E-Mail-Adresse.
func (s *MemoryStore) GetUserByEmail(_ context.Context, email string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userID, exists := s.emailIndex[email]
	if !exists {
		return nil, ErrNotFound
	}

	user := s.users[userID]
	return &user, nil
}

// EnsureDevOrg ist ein Noop für MemoryStore (keine DB needed).
func (s *MemoryStore) EnsureDevOrg(_ context.Context, _ string) error {
	return nil
}
