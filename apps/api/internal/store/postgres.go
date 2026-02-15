package store

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/max-cloud/api/internal/auth"
	"github.com/max-cloud/shared/pkg/models"
)

// UpdateStatus setzt Status und URL eines Services.
func (s *PostgresStore) UpdateStatus(ctx context.Context, id string, status models.ServiceStatus, url string) error {
	result, err := s.pool.Exec(ctx,
		`UPDATE services SET status = $1, url = COALESCE(NULLIF($2, ''), url), updated_at = NOW() WHERE id = $3`,
		string(status), url, id,
	)
	if err != nil {
		return fmt.Errorf("updating service status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

//go:embed migrations/*.sql
var migrationsFS embed.FS

// PostgresStore implementiert ServiceStore mit PostgreSQL als Backend.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgres erstellt einen neuen PostgresStore, verbindet sich mit der DB und führt Migrationen aus.
func NewPostgres(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	if err := runMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	return &PostgresStore{pool: pool}, nil
}

// Close schliesst den Connection-Pool.
func (s *PostgresStore) Close() {
	s.pool.Close()
}

// runMigrations führt alle SQL-Dateien aus dem migrations-Verzeichnis in alphabetischer Reihenfolge aus.
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Migration-Tabelle erstellen falls nicht vorhanden
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("creating migration table: %w", err)
	}

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("reading migrations dir: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		version := entry.Name()

		// Prüfen ob Migration bereits ausgeführt
		var exists bool
		err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`,
			version,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("checking migration status %s: %w", version, err)
		}

		if exists {
			continue
		}

		// Migration ausführen
		data, err := fs.ReadFile(migrationsFS, "migrations/"+version)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", version, err)
		}

		if _, err := pool.Exec(ctx, string(data)); err != nil {
			return fmt.Errorf("running migration %s: %w", version, err)
		}

		// Migration als ausgeführt markieren
		if _, err := pool.Exec(ctx,
			`INSERT INTO schema_migrations (version) VALUES ($1)`,
			version,
		); err != nil {
			return fmt.Errorf("recording migration %s: %w", version, err)
		}
	}
	return nil
}

// Create fügt einen neuen Service ein. UUID und Timestamps werden von PostgreSQL generiert.
func (s *PostgresStore) Create(ctx context.Context, req models.DeployRequest) (models.Service, error) {
	envJSON, err := json.Marshal(req.EnvVars)
	if err != nil {
		return models.Service{}, fmt.Errorf("marshaling env_vars: %w", err)
	}
	if req.EnvVars == nil {
		envJSON = []byte("{}")
	}

	commandJSON, err := json.Marshal(req.Command)
	if err != nil {
		return models.Service{}, fmt.Errorf("marshaling command: %w", err)
	}
	if len(req.Command) == 0 {
		commandJSON = []byte("[]")
	}

	argsJSON, err := json.Marshal(req.Args)
	if err != nil {
		return models.Service{}, fmt.Errorf("marshaling args: %w", err)
	}
	if len(req.Args) == 0 {
		argsJSON = []byte("[]")
	}

	var orgIDParam any
	if orgID, ok := auth.OrgIDFromContext(ctx); ok {
		orgIDParam = orgID
	}

	var svc models.Service
	var envBytes, commandBytes, argsBytes []byte
	var orgID *string
	err = s.pool.QueryRow(ctx,
		`INSERT INTO services (name, image, status, url, env_vars, org_id, port, command, args)
		 VALUES ($1, $2, 'pending', '', $3, $4, $5, $6, $7)
		 RETURNING id, name, image, status, url, env_vars, min_scale, max_scale, created_at, updated_at, org_id, port, command, args`,
		req.Name, req.Image, envJSON, orgIDParam, req.Port, commandJSON, argsJSON,
	).Scan(
		&svc.ID, &svc.Name, &svc.Image, &svc.Status, &svc.URL,
		&envBytes, &svc.MinScale, &svc.MaxScale, &svc.CreatedAt, &svc.UpdatedAt, &orgID,
		&svc.Port, &commandBytes, &argsBytes,
	)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return models.Service{}, ErrDuplicateService
		}
		return models.Service{}, fmt.Errorf("inserting service: %w", err)
	}

	if orgID != nil {
		svc.OrgID = *orgID
	}

	if err := json.Unmarshal(envBytes, &svc.EnvVars); err != nil {
		return models.Service{}, fmt.Errorf("unmarshaling env_vars: %w", err)
	}

	if err := json.Unmarshal(commandBytes, &svc.Command); err != nil {
		return models.Service{}, fmt.Errorf("unmarshaling command: %w", err)
	}

	if err := json.Unmarshal(argsBytes, &svc.Args); err != nil {
		return models.Service{}, fmt.Errorf("unmarshaling args: %w", err)
	}

	return svc, nil
}

// Get gibt einen Service anhand seiner ID zurück. Gibt ErrNotFound zurück, wenn nicht vorhanden.
func (s *PostgresStore) Get(ctx context.Context, id string) (models.Service, error) {
	query := `SELECT id, name, image, status, url, env_vars, min_scale, max_scale, created_at, updated_at, org_id, port, command, args
		 FROM services WHERE id = $1`
	args := []any{id}

	if orgID, ok := auth.OrgIDFromContext(ctx); ok {
		query = `SELECT id, name, image, status, url, env_vars, min_scale, max_scale, created_at, updated_at, org_id, port, command, args
			 FROM services WHERE id = $1 AND org_id = $2`
		args = append(args, orgID)
	}

	var svc models.Service
	var envBytes, commandBytes, argsBytes []byte
	var orgID *string
	err := s.pool.QueryRow(ctx, query, args...).Scan(
		&svc.ID, &svc.Name, &svc.Image, &svc.Status, &svc.URL,
		&envBytes, &svc.MinScale, &svc.MaxScale, &svc.CreatedAt, &svc.UpdatedAt, &orgID,
		&svc.Port, &commandBytes, &argsBytes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Service{}, ErrNotFound
		}
		return models.Service{}, fmt.Errorf("querying service: %w", err)
	}

	if orgID != nil {
		svc.OrgID = *orgID
	}

	if err := json.Unmarshal(envBytes, &svc.EnvVars); err != nil {
		return models.Service{}, fmt.Errorf("unmarshaling env_vars: %w", err)
	}

	if err := json.Unmarshal(commandBytes, &svc.Command); err != nil {
		return models.Service{}, fmt.Errorf("unmarshaling command: %w", err)
	}

	if err := json.Unmarshal(argsBytes, &svc.Args); err != nil {
		return models.Service{}, fmt.Errorf("unmarshaling args: %w", err)
	}

	return svc, nil
}

// GetByName gibt einen Service anhand seines Namens zurück.
func (s *PostgresStore) GetByName(ctx context.Context, name string) (models.Service, error) {
	query := `SELECT id, name, image, status, url, env_vars, min_scale, max_scale, created_at, updated_at, org_id, port, command, args
		 FROM services WHERE name = $1`
	args := []any{name}

	if orgID, ok := auth.OrgIDFromContext(ctx); ok {
		query = `SELECT id, name, image, status, url, env_vars, min_scale, max_scale, created_at, updated_at, org_id
			 FROM services WHERE name = $1 AND org_id = $2`
		args = append(args, orgID)
	}

	var svc models.Service
	var envBytes, commandBytes, argsBytes []byte
	var orgID *string
	err := s.pool.QueryRow(ctx, query, args...).Scan(
		&svc.ID, &svc.Name, &svc.Image, &svc.Status, &svc.URL,
		&envBytes, &svc.MinScale, &svc.MaxScale, &svc.CreatedAt, &svc.UpdatedAt, &orgID,
		&svc.Port, &commandBytes, &argsBytes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Service{}, ErrNotFound
		}
		return models.Service{}, fmt.Errorf("querying service by name: %w", err)
	}

	if orgID != nil {
		svc.OrgID = *orgID
	}

	if err := json.Unmarshal(envBytes, &svc.EnvVars); err != nil {
		return models.Service{}, fmt.Errorf("unmarshaling env_vars: %w", err)
	}

	if err := json.Unmarshal(commandBytes, &svc.Command); err != nil {
		return models.Service{}, fmt.Errorf("unmarshaling command: %w", err)
	}

	if err := json.Unmarshal(argsBytes, &svc.Args); err != nil {
		return models.Service{}, fmt.Errorf("unmarshaling args: %w", err)
	}

	return svc, nil
}

// List gibt alle Services zurück.
func (s *PostgresStore) List(ctx context.Context) ([]models.Service, error) {
	query := `SELECT id, name, image, status, url, env_vars, min_scale, max_scale, created_at, updated_at, org_id, port, command, args
		 FROM services`
	var args []any

	if orgID, ok := auth.OrgIDFromContext(ctx); ok {
		query += ` WHERE org_id = $1`
		args = append(args, orgID)
	}
	query += ` ORDER BY created_at`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying services: %w", err)
	}
	defer rows.Close()

	var services []models.Service
	for rows.Next() {
		var svc models.Service
		var envBytes, commandBytes, argsBytes []byte
		var orgID *string
		if err := rows.Scan(
			&svc.ID, &svc.Name, &svc.Image, &svc.Status, &svc.URL,
			&envBytes, &svc.MinScale, &svc.MaxScale, &svc.CreatedAt, &svc.UpdatedAt, &orgID,
			&svc.Port, &commandBytes, &argsBytes,
		); err != nil {
			return nil, fmt.Errorf("scanning service: %w", err)
		}
		if orgID != nil {
			svc.OrgID = *orgID
		}
		if err := json.Unmarshal(envBytes, &svc.EnvVars); err != nil {
			return nil, fmt.Errorf("unmarshaling env_vars: %w", err)
		}
		if err := json.Unmarshal(commandBytes, &svc.Command); err != nil {
			return nil, fmt.Errorf("unmarshaling command: %w", err)
		}
		if err := json.Unmarshal(argsBytes, &svc.Args); err != nil {
			return nil, fmt.Errorf("unmarshaling args: %w", err)
		}
		services = append(services, svc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating services: %w", err)
	}

	if services == nil {
		services = []models.Service{}
	}

	return services, nil
}

// Delete entfernt einen Service anhand seiner ID. Gibt ErrNotFound zurück, wenn nicht vorhanden.
func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM services WHERE id = $1`
	args := []any{id}

	if orgID, ok := auth.OrgIDFromContext(ctx); ok {
		query = `DELETE FROM services WHERE id = $1 AND org_id = $2`
		args = append(args, orgID)
	}

	result, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("deleting service: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
