// Package sqlite provides SQLite-based storage implementations for locdoc services.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// DB represents a SQLite database connection.
type DB struct {
	db   *sql.DB
	path string
}

// NewDB creates a new DB instance with the given path.
// Use ":memory:" for an in-memory database.
func NewDB(path string) *DB {
	return &DB{path: path}
}

// Open opens the database connection and creates the schema if needed.
func (db *DB) Open() error {
	conn, err := sql.Open("sqlite3", db.path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// SQLite only supports one writer at a time, so limit to one connection.
	conn.SetMaxOpenConns(1)

	// Verify connection
	if err := conn.Ping(); err != nil {
		conn.Close()
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set busy timeout to wait 5 seconds before failing on lock contention.
	// This prevents immediate "database is locked" errors.
	if _, err := conn.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		conn.Close()
		return fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Enable WAL mode for file-based databases for better write performance.
	// WAL is ~7x faster for writes and allows concurrent reads during writes.
	// Trade-off: creates additional -wal and -shm files alongside the database.
	// Note: WAL mode is not supported for in-memory databases.
	if db.path != ":memory:" {
		if _, err := conn.Exec("PRAGMA journal_mode = WAL"); err != nil {
			conn.Close()
			return fmt.Errorf("failed to enable WAL mode: %w", err)
		}
	}

	// Enable foreign key constraints
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		conn.Close()
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	db.db = conn

	// Create schema
	if err := db.createSchema(); err != nil {
		conn.Close()
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	if db.db != nil {
		return db.db.Close()
	}
	return nil
}

// QueryRowContext executes a query that returns a single row.
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return db.db.QueryRowContext(ctx, query, args...)
}

// QueryContext executes a query that returns rows.
func (db *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.db.QueryContext(ctx, query, args...)
}

// ExecContext executes a statement that doesn't return rows.
func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.db.ExecContext(ctx, query, args...)
}

// Stats returns database statistics.
func (db *DB) Stats() sql.DBStats {
	return db.db.Stats()
}

// createSchema creates the database tables if they don't exist.
func (db *DB) createSchema() error {
	schema := `
		CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			source_url TEXT NOT NULL,
			local_path TEXT NOT NULL DEFAULT '',
			filter TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS documents (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			file_path TEXT NOT NULL DEFAULT '',
			source_url TEXT NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL DEFAULT '',
			content_hash TEXT NOT NULL DEFAULT '',
			position INTEGER NOT NULL DEFAULT 0,
			fetched_at TEXT NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_documents_project_id ON documents(project_id);
		CREATE INDEX IF NOT EXISTS idx_documents_source_url ON documents(source_url);
	`

	_, err := db.db.Exec(schema)
	return err
}
