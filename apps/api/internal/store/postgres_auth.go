package store

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/max-cloud/shared/pkg/models"
)

// Register erstellt User, Organisation, Membership und initialen API-Key in einer Transaktion.
func (s *PostgresStore) Register(ctx context.Context, email, orgName string) (models.User, models.Organization, string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.User{}, models.Organization{}, "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Organisation anlegen
	var org models.Organization
	err = tx.QueryRow(ctx,
		`INSERT INTO organizations (name) VALUES ($1) RETURNING id, name, created_at`,
		orgName,
	).Scan(&org.ID, &org.Name, &org.CreatedAt)
	if err != nil {
		if isDuplicateError(err) {
			return models.User{}, models.Organization{}, "", ErrDuplicateOrg
		}
		return models.User{}, models.Organization{}, "", fmt.Errorf("insert org: %w", err)
	}

	// User anlegen
	var user models.User
	err = tx.QueryRow(ctx,
		`INSERT INTO users (email) VALUES ($1) RETURNING id, email, created_at`,
		email,
	).Scan(&user.ID, &user.Email, &user.CreatedAt)
	if err != nil {
		if isDuplicateError(err) {
			return models.User{}, models.Organization{}, "", ErrDuplicateEmail
		}
		return models.User{}, models.Organization{}, "", fmt.Errorf("insert user: %w", err)
	}

	// Membership
	if _, err := tx.Exec(ctx,
		`INSERT INTO org_members (org_id, user_id, role) VALUES ($1, $2, 'admin')`,
		org.ID, user.ID,
	); err != nil {
		return models.User{}, models.Organization{}, "", fmt.Errorf("insert membership: %w", err)
	}

	// API-Key
	rawKey, keyHash, prefix, err := generateAPIKey()
	if err != nil {
		return models.User{}, models.Organization{}, "", err
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO api_keys (key_hash, prefix, name, org_id, user_id) VALUES ($1, $2, 'default', $3, $4)`,
		keyHash, prefix, org.ID, user.ID,
	); err != nil {
		return models.User{}, models.Organization{}, "", fmt.Errorf("insert api key: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.User{}, models.Organization{}, "", fmt.Errorf("commit: %w", err)
	}

	return user, org, rawKey, nil
}

// ValidateAPIKey prüft einen API-Key via Prefix-Lookup und Hash-Vergleich.
func (s *PostgresStore) ValidateAPIKey(ctx context.Context, rawKey string) (*models.APIKeyInfo, error) {
	prefix, err := extractPrefix(rawKey)
	if err != nil {
		return nil, ErrKeyNotFound
	}

	hash := hashAPIKey(rawKey)

	rows, err := s.pool.Query(ctx,
		`SELECT id, key_hash, prefix, name, org_id, user_id, created_at, expires_at, last_used_at
		 FROM api_keys WHERE prefix = $1`,
		prefix,
	)
	if err != nil {
		return nil, fmt.Errorf("querying api keys: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var info models.APIKeyInfo
		var dbHash string
		if err := rows.Scan(
			&info.ID, &dbHash, &info.Prefix, &info.Name,
			&info.OrgID, &info.UserID, &info.CreatedAt, &info.ExpiresAt, &info.LastUsedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning api key: %w", err)
		}

		if subtle.ConstantTimeCompare([]byte(dbHash), []byte(hash)) == 1 {
			if info.ExpiresAt != nil && info.ExpiresAt.Before(time.Now()) {
				return nil, ErrKeyNotFound
			}
			return &info, nil
		}
	}

	return nil, ErrKeyNotFound
}

// CreateAPIKey erstellt einen neuen API-Key.
func (s *PostgresStore) CreateAPIKey(ctx context.Context, orgID, userID, name string) (string, *models.APIKeyInfo, error) {
	rawKey, keyHash, prefix, err := generateAPIKey()
	if err != nil {
		return "", nil, err
	}

	var info models.APIKeyInfo
	err = s.pool.QueryRow(ctx,
		`INSERT INTO api_keys (key_hash, prefix, name, org_id, user_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, prefix, name, org_id, user_id, created_at, expires_at, last_used_at`,
		keyHash, prefix, name, orgID, userID,
	).Scan(&info.ID, &info.Prefix, &info.Name, &info.OrgID, &info.UserID,
		&info.CreatedAt, &info.ExpiresAt, &info.LastUsedAt)
	if err != nil {
		return "", nil, fmt.Errorf("inserting api key: %w", err)
	}

	return rawKey, &info, nil
}

// ListAPIKeys gibt alle API-Keys einer Organisation zurück.
func (s *PostgresStore) ListAPIKeys(ctx context.Context, orgID string) ([]models.APIKeyInfo, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, prefix, name, org_id, user_id, created_at, expires_at, last_used_at
		 FROM api_keys WHERE org_id = $1 ORDER BY created_at`,
		orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying api keys: %w", err)
	}
	defer rows.Close()

	var keys []models.APIKeyInfo
	for rows.Next() {
		var info models.APIKeyInfo
		if err := rows.Scan(
			&info.ID, &info.Prefix, &info.Name, &info.OrgID, &info.UserID,
			&info.CreatedAt, &info.ExpiresAt, &info.LastUsedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning api key: %w", err)
		}
		keys = append(keys, info)
	}
	if keys == nil {
		keys = []models.APIKeyInfo{}
	}
	return keys, nil
}

// DeleteAPIKey löscht einen API-Key.
func (s *PostgresStore) DeleteAPIKey(ctx context.Context, orgID, keyID string) error {
	result, err := s.pool.Exec(ctx,
		`DELETE FROM api_keys WHERE id = $1 AND org_id = $2`,
		keyID, orgID,
	)
	if err != nil {
		return fmt.Errorf("deleting api key: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrKeyNotFound
	}
	return nil
}

// GetAuthInfo gibt Informationen über User, Organisation und Rolle zurück.
func (s *PostgresStore) GetAuthInfo(ctx context.Context, orgID, userID string) (*models.AuthInfo, error) {
	var info models.AuthInfo
	var role string
	err := s.pool.QueryRow(ctx,
		`SELECT u.id, u.email, u.created_at,
		        o.id, o.name, o.created_at,
		        m.role
		 FROM users u
		 JOIN org_members m ON m.user_id = u.id
		 JOIN organizations o ON o.id = m.org_id
		 WHERE o.id = $1 AND u.id = $2`,
		orgID, userID,
	).Scan(
		&info.User.ID, &info.User.Email, &info.User.CreatedAt,
		&info.Organization.ID, &info.Organization.Name, &info.Organization.CreatedAt,
		&role,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("querying auth info: %w", err)
	}
	info.Role = models.OrgRole(role)
	return &info, nil
}

// UpdateAPIKeyLastUsed aktualisiert den last_used_at Timestamp.
func (s *PostgresStore) UpdateAPIKeyLastUsed(ctx context.Context, keyID string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`,
		keyID,
	)
	if err != nil {
		return fmt.Errorf("updating last_used_at: %w", err)
	}
	return nil
}

// isDuplicateError prüft auf PostgreSQL unique constraint violation.
func isDuplicateError(err error) bool {
	return err != nil && (errors.Is(err, pgx.ErrNoRows) == false) &&
		(len(err.Error()) > 0 && containsDuplicateCode(err.Error()))
}

func containsDuplicateCode(s string) bool {
	// PostgreSQL error code 23505 = unique_violation
	return contains(s, "23505") || contains(s, "duplicate key")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// CreateInvite erstellt eine neue Einladung für eine Organisation.
func (s *PostgresStore) CreateInvite(ctx context.Context, orgID, email string, role models.OrgRole, invitedBy string, expiresAt time.Time) (models.Invitation, string, error) {
	// Prüfen ob User bereits Mitglied der Org ist
	var memberCount int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM org_members m
		 JOIN users u ON u.id = m.user_id
		 WHERE m.org_id = $1 AND u.email = $2`,
		orgID, email,
	).Scan(&memberCount)
	if err != nil {
		return models.Invitation{}, "", fmt.Errorf("checking membership: %w", err)
	}
	if memberCount > 0 {
		return models.Invitation{}, "", ErrAlreadyMember
	}

	rawToken, tokenHash, tokenPrefix, err := generateInviteToken()
	if err != nil {
		return models.Invitation{}, "", err
	}

	var invite models.Invitation
	var orgName string
	err = s.pool.QueryRow(ctx,
		`INSERT INTO invitations (org_id, email, role, token_hash, token_prefix, invited_by, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, org_id, email, role, status, invited_by, expires_at, created_at`,
		orgID, email, string(role), tokenHash, tokenPrefix, invitedBy, expiresAt,
	).Scan(&invite.ID, &invite.OrgID, &invite.Email, &invite.Role, &invite.Status,
		&invite.InvitedBy, &invite.ExpiresAt, &invite.CreatedAt)
	if err != nil {
		return models.Invitation{}, "", fmt.Errorf("inserting invitation: %w", err)
	}

	// Org-Name abrufen
	err = s.pool.QueryRow(ctx, `SELECT name FROM organizations WHERE id = $1`, orgID).Scan(&orgName)
	if err != nil {
		return models.Invitation{}, "", fmt.Errorf("querying org name: %w", err)
	}
	invite.OrgName = orgName

	return invite, rawToken, nil
}

// AcceptInvite nimmt eine Einladung an, erstellt ggf. den User und fügt ihn zur Org hinzu.
func (s *PostgresStore) AcceptInvite(ctx context.Context, rawToken string) (models.User, models.Organization, models.OrgRole, string, error) {
	prefix, err := extractInvitePrefix(rawToken)
	if err != nil {
		return models.User{}, models.Organization{}, "", "", ErrInviteNotFound
	}

	tokenHash := hashInviteToken(rawToken)

	// Token-Candidates per Prefix laden
	rows, err := s.pool.Query(ctx,
		`SELECT id, org_id, email, role, status, token_hash, expires_at
		 FROM invitations WHERE token_prefix = $1`,
		prefix,
	)
	if err != nil {
		return models.User{}, models.Organization{}, "", "", fmt.Errorf("querying invitations: %w", err)
	}
	defer rows.Close()

	var inviteID, orgID, email, dbHash string
	var role models.OrgRole
	var status models.InviteStatus
	var expiresAt time.Time
	var found bool

	for rows.Next() {
		var candidateHash string
		var candidateID, candidateOrgID, candidateEmail string
		var candidateRole models.OrgRole
		var candidateStatus models.InviteStatus
		var candidateExpires time.Time

		if err := rows.Scan(&candidateID, &candidateOrgID, &candidateEmail, &candidateRole,
			&candidateStatus, &candidateHash, &candidateExpires); err != nil {
			return models.User{}, models.Organization{}, "", "", fmt.Errorf("scanning invitation: %w", err)
		}
		if subtle.ConstantTimeCompare([]byte(candidateHash), []byte(tokenHash)) == 1 {
			inviteID = candidateID
			orgID = candidateOrgID
			email = candidateEmail
			role = candidateRole
			status = candidateStatus
			dbHash = candidateHash
			expiresAt = candidateExpires
			found = true
			break
		}
	}
	rows.Close()
	_ = dbHash

	if !found {
		return models.User{}, models.Organization{}, "", "", ErrInviteNotFound
	}

	if status != models.InviteStatusPending {
		return models.User{}, models.Organization{}, "", "", ErrInviteNotFound
	}

	if time.Now().After(expiresAt) {
		s.pool.Exec(ctx, `UPDATE invitations SET status = 'expired' WHERE id = $1`, inviteID)
		return models.User{}, models.Organization{}, "", "", ErrInviteExpired
	}

	// Transaktion: User finden/erstellen, Membership, API-Key, Invite aktualisieren
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.User{}, models.Organization{}, "", "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// User finden oder erstellen
	var user models.User
	err = tx.QueryRow(ctx,
		`SELECT id, email, created_at FROM users WHERE email = $1`, email,
	).Scan(&user.ID, &user.Email, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = tx.QueryRow(ctx,
				`INSERT INTO users (email) VALUES ($1) RETURNING id, email, created_at`, email,
			).Scan(&user.ID, &user.Email, &user.CreatedAt)
			if err != nil {
				return models.User{}, models.Organization{}, "", "", fmt.Errorf("insert user: %w", err)
			}
		} else {
			return models.User{}, models.Organization{}, "", "", fmt.Errorf("query user: %w", err)
		}
	}

	// Membership anlegen
	if _, err := tx.Exec(ctx,
		`INSERT INTO org_members (org_id, user_id, role) VALUES ($1, $2, $3)`,
		orgID, user.ID, string(role),
	); err != nil {
		return models.User{}, models.Organization{}, "", "", fmt.Errorf("insert membership: %w", err)
	}

	// API-Key generieren
	rawKey, keyHash, keyPrefix, err := generateAPIKey()
	if err != nil {
		return models.User{}, models.Organization{}, "", "", err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO api_keys (key_hash, prefix, name, org_id, user_id) VALUES ($1, $2, 'default', $3, $4)`,
		keyHash, keyPrefix, orgID, user.ID,
	); err != nil {
		return models.User{}, models.Organization{}, "", "", fmt.Errorf("insert api key: %w", err)
	}

	// Invite als accepted markieren
	if _, err := tx.Exec(ctx,
		`UPDATE invitations SET status = 'accepted' WHERE id = $1`, inviteID,
	); err != nil {
		return models.User{}, models.Organization{}, "", "", fmt.Errorf("update invitation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.User{}, models.Organization{}, "", "", fmt.Errorf("commit: %w", err)
	}

	// Organisation laden
	var org models.Organization
	err = s.pool.QueryRow(ctx,
		`SELECT id, name, created_at FROM organizations WHERE id = $1`, orgID,
	).Scan(&org.ID, &org.Name, &org.CreatedAt)
	if err != nil {
		return models.User{}, models.Organization{}, "", "", fmt.Errorf("query org: %w", err)
	}

	return user, org, role, rawKey, nil
}

// ListInvites gibt alle pending Einladungen einer Organisation zurück.
func (s *PostgresStore) ListInvites(ctx context.Context, orgID string) ([]models.Invitation, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT i.id, i.org_id, o.name, i.email, i.role, i.status, i.invited_by, i.expires_at, i.created_at
		 FROM invitations i
		 JOIN organizations o ON o.id = i.org_id
		 WHERE i.org_id = $1 AND i.status = 'pending'
		 ORDER BY i.created_at`,
		orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying invitations: %w", err)
	}
	defer rows.Close()

	var invites []models.Invitation
	for rows.Next() {
		var inv models.Invitation
		if err := rows.Scan(&inv.ID, &inv.OrgID, &inv.OrgName, &inv.Email, &inv.Role,
			&inv.Status, &inv.InvitedBy, &inv.ExpiresAt, &inv.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning invitation: %w", err)
		}
		invites = append(invites, inv)
	}
	if invites == nil {
		invites = []models.Invitation{}
	}
	return invites, nil
}

// RevokeInvite widerruft eine Einladung.
func (s *PostgresStore) RevokeInvite(ctx context.Context, orgID, inviteID string) error {
	result, err := s.pool.Exec(ctx,
		`UPDATE invitations SET status = 'revoked' WHERE id = $1 AND org_id = $2 AND status = 'pending'`,
		inviteID, orgID,
	)
	if err != nil {
		return fmt.Errorf("revoking invitation: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrInviteNotFound
	}
	return nil
}

// GetUserByEmail sucht einen User anhand seiner E-Mail-Adresse.
func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, created_at FROM users WHERE email = $1`, email,
	).Scan(&user.ID, &user.Email, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("querying user by email: %w", err)
	}
	return &user, nil
}

// EnsureDevOrg creates a default development organization if it doesn't exist.
// This is used in DEV_MODE when DATABASE_URL is set.
func (s *PostgresStore) EnsureDevOrg(ctx context.Context, devOrgID string) error {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM organizations WHERE id = $1)`,
		devOrgID,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("checking dev org: %w", err)
	}
	if exists {
		return nil
	}

	_, err = s.pool.Exec(ctx,
		`INSERT INTO organizations (id, name) VALUES ($1, 'dev-org')`,
		devOrgID,
	)
	if err != nil && !isDuplicateError(err) {
		return fmt.Errorf("creating dev org: %w", err)
	}
	return nil
}
