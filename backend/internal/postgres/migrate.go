// Package postgres implements storage on top of PostgreSQL.
package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	"github.com/pressly/goose/v3"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/migrations"
)

// OpenSQL opens a database/sql connection pool using the pgx driver.
// It is used for migrations; repositories use pgxpool directly.
func OpenSQL(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return db, nil
}

// NewMigrator returns a goose provider over the embedded migrations.
// Tests use it to bring a fresh database to the current schema.
func NewMigrator(db *sql.DB) (*goose.Provider, error) {
	p, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		return nil, fmt.Errorf("create migrator: %w", err)
	}
	return p, nil
}
