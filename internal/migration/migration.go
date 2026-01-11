package migration

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

//go:embed sql/*.sql
var migrationFiles embed.FS

// Migration represents a single database migration
type Migration struct {
	Version     int
	Name        string
	SQL         string
	AppliedAt   *time.Time
	Description string
}

// Manager handles database migrations
type Manager struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewManager creates a new migration manager
func NewManager(db *sql.DB, logger *logrus.Logger) *Manager {
	if logger == nil {
		logger = logrus.New()
	}
	return &Manager{
		db:     db,
		logger: logger,
	}
}

// Initialize creates the schema_migrations table if it doesn't exist
func (m *Manager) Initialize(ctx context.Context) error {
	m.logger.Info("Initializing migration system...")

	createTableSQL := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			checksum VARCHAR(64) NOT NULL,
			execution_time_ms INT NOT NULL,
			INDEX idx_applied_at (applied_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	if _, err := m.db.ExecContext(ctx, createTableSQL); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	m.logger.Info("Migration system initialized")
	return nil
}

// GetAppliedMigrations returns all applied migrations
func (m *Manager) GetAppliedMigrations(ctx context.Context) (map[int]*Migration, error) {
	query := `
		SELECT version, name, description, applied_at, checksum
		FROM schema_migrations
		ORDER BY version
	`

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]*Migration)
	for rows.Next() {
		var migration Migration
		var checksum string
		if err := rows.Scan(&migration.Version, &migration.Name, &migration.Description, &migration.AppliedAt, &checksum); err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		applied[migration.Version] = &migration
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migration rows: %w", err)
	}

	return applied, nil
}

// GetPendingMigrations returns migrations that haven't been applied yet
func (m *Manager) GetPendingMigrations(ctx context.Context) ([]*Migration, error) {
	// Get all available migrations
	available, err := m.loadAvailableMigrations()
	if err != nil {
		return nil, err
	}

	// Get applied migrations
	applied, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	// Find pending migrations
	var pending []*Migration
	for _, migration := range available {
		if _, exists := applied[migration.Version]; !exists {
			pending = append(pending, migration)
		}
	}

	// Sort by version
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Version < pending[j].Version
	})

	return pending, nil
}

// Migrate runs all pending migrations
func (m *Manager) Migrate(ctx context.Context) error {
	m.logger.Info("Starting database migration...")

	// Initialize migration system
	if err := m.Initialize(ctx); err != nil {
		return err
	}

	// Get pending migrations
	pending, err := m.GetPendingMigrations(ctx)
	if err != nil {
		return err
	}

	if len(pending) == 0 {
		m.logger.Info("No pending migrations")
		return nil
	}

	m.logger.Infof("Found %d pending migration(s)", len(pending))

	// Apply each migration
	for _, migration := range pending {
		if err := m.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}
	}

	m.logger.Info("All migrations completed successfully")
	return nil
}

// MigrateToVersion migrates to a specific version
func (m *Manager) MigrateToVersion(ctx context.Context, targetVersion int) error {
	m.logger.Infof("Migrating to version %d...", targetVersion)

	// Initialize migration system
	if err := m.Initialize(ctx); err != nil {
		return err
	}

	// Get pending migrations
	pending, err := m.GetPendingMigrations(ctx)
	if err != nil {
		return err
	}

	// Filter migrations up to target version
	var toApply []*Migration
	for _, migration := range pending {
		if migration.Version <= targetVersion {
			toApply = append(toApply, migration)
		}
	}

	if len(toApply) == 0 {
		m.logger.Info("No migrations to apply")
		return nil
	}

	m.logger.Infof("Applying %d migration(s)", len(toApply))

	// Apply each migration
	for _, migration := range toApply {
		if err := m.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}
	}

	m.logger.Infof("Successfully migrated to version %d", targetVersion)
	return nil
}

// GetCurrentVersion returns the current migration version
func (m *Manager) GetCurrentVersion(ctx context.Context) (int, error) {
	query := `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`

	var version int
	if err := m.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}

	return version, nil
}

// Status returns the current migration status
func (m *Manager) Status(ctx context.Context) (string, error) {
	// Initialize if needed
	if err := m.Initialize(ctx); err != nil {
		return "", err
	}

	currentVersion, err := m.GetCurrentVersion(ctx)
	if err != nil {
		return "", err
	}

	pending, err := m.GetPendingMigrations(ctx)
	if err != nil {
		return "", err
	}

	applied, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return "", err
	}

	status := fmt.Sprintf("Current version: %d\n", currentVersion)
	status += fmt.Sprintf("Applied migrations: %d\n", len(applied))
	status += fmt.Sprintf("Pending migrations: %d\n", len(pending))

	if len(pending) > 0 {
		status += "\nPending migrations:\n"
		for _, migration := range pending {
			status += fmt.Sprintf("  - Version %d: %s\n", migration.Version, migration.Name)
		}
	}

	return status, nil
}

// applyMigration applies a single migration
func (m *Manager) applyMigration(ctx context.Context, migration *Migration) error {
	m.logger.Infof("Applying migration %d: %s", migration.Version, migration.Name)

	startTime := time.Now()

	// Start transaction
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Calculate execution time
	executionTime := time.Since(startTime).Milliseconds()

	// Calculate checksum
	checksum := calculateChecksum(migration.SQL)

	// Record migration
	recordSQL := `
		INSERT INTO schema_migrations (version, name, description, checksum, execution_time_ms)
		VALUES (?, ?, ?, ?, ?)
	`
	if _, err := tx.ExecContext(ctx, recordSQL, migration.Version, migration.Name, migration.Description, checksum, executionTime); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	m.logger.Infof("Migration %d applied successfully (took %dms)", migration.Version, executionTime)
	return nil
}

// loadAvailableMigrations loads all available migrations from embedded files
func (m *Manager) loadAvailableMigrations() ([]*Migration, error) {
	entries, err := migrationFiles.ReadDir("sql")
	if err != nil {
		return nil, fmt.Errorf("failed to read migration directory: %w", err)
	}

	var migrations []*Migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		// Parse migration file
		migration, err := m.parseMigrationFile(entry.Name())
		if err != nil {
			m.logger.Warnf("Failed to parse migration file %s: %v", entry.Name(), err)
			continue
		}

		migrations = append(migrations, migration)
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// parseMigrationFile parses a migration file and extracts metadata
func (m *Manager) parseMigrationFile(filename string) (*Migration, error) {
	// Read file content
	content, err := migrationFiles.ReadFile("sql/" + filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	sql := string(content)

	// Parse metadata from comments
	migration := &Migration{
		SQL: sql,
	}

	lines := strings.Split(sql, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "--") {
			break
		}

		// Remove comment prefix
		line = strings.TrimPrefix(line, "--")
		line = strings.TrimSpace(line)

		// Parse metadata
		if strings.HasPrefix(line, "Version:") {
			fmt.Sscanf(line, "Version: %d", &migration.Version)
		} else if strings.HasPrefix(line, "Name:") {
			migration.Name = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
		} else if strings.HasPrefix(line, "Description:") {
			migration.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
		}
	}

	if migration.Version == 0 {
		return nil, fmt.Errorf("migration version not found in file")
	}

	if migration.Name == "" {
		migration.Name = filename
	}

	return migration, nil
}

// calculateChecksum calculates a simple checksum for migration content
func calculateChecksum(content string) string {
	// Simple checksum using length and first/last characters
	// In production, use a proper hash function like SHA256
	if len(content) == 0 {
		return "empty"
	}
	return fmt.Sprintf("%d-%c-%c", len(content), content[0], content[len(content)-1])
}
