package tursoutil

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
)

type Migrator struct {
	db          *sqlx.DB
	service     string
	logger      *slog.Logger
	migrationFS embed.FS
}

func NewMigrator(db *sqlx.DB, service string, logger *slog.Logger, migrationFS embed.FS) *Migrator {
	return &Migrator{
		db:          db,
		service:     service,
		logger:      logger,
		migrationFS: migrationFS,
	}
}

func (m *Migrator) Migrate(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			service TEXT NOT NULL,
			version INTEGER NOT NULL,
			applied_at TEXT NOT NULL DEFAULT (datetime('now')),
			PRIMARY KEY (service, version)
		);
	`)
	if err != nil {
		return fmt.Errorf("creating schema_migrations: %w", err)
	}

	applied, err := m.appliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("reading applied versions: %w", err)
	}
	m.logger.Debug("applied migrations", slog.String("service", m.service), slog.Int("count", len(applied)))

	files, err := m.migrationFiles()
	if err != nil {
		return fmt.Errorf("reading migration files: %w", err)
	}

	for _, mf := range files {
		if applied[mf.version] {
			continue
		}
		m.logger.Debug("applying migration", slog.String("service", m.service), slog.Int("version", mf.version), slog.String("path", mf.path))

		content, err := fs.ReadFile(m.migrationFS, mf.path)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", mf.path, err)
		}

		tx, err := m.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx for migration %d: %w", mf.version, err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("executing migration %d: %w", mf.version, err)
		}

		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (service, version) VALUES (?, ?);", m.service, mf.version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("recording migration %d: %w", mf.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", mf.version, err)
		}
	}

	return nil
}

type migrationFile struct {
	version int
	path    string
}

func (m *Migrator) appliedVersions(ctx context.Context) (map[int]bool, error) {
	rows, err := m.db.QueryContext(ctx, "SELECT version FROM schema_migrations WHERE service = ?", m.service)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]bool)
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		result[v] = true
	}
	return result, rows.Err()
}

func (m *Migrator) migrationFiles() ([]migrationFile, error) {
	entries, err := fs.ReadDir(m.migrationFS, "migrations")
	if err != nil {
		return nil, err
	}

	var files []migrationFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) < 2 {
			continue
		}
		version, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		files = append(files, migrationFile{
			version: version,
			path:    "migrations/" + entry.Name(),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].version < files[j].version
	})

	return files, nil
}
