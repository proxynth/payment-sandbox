package migrations

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed sql/*.sql
var files embed.FS

func Up(db *sql.DB) error {
	goose.SetBaseFS(files)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.Up(db, "sql"); err != nil {
		return fmt.Errorf("apply database migrations: %w", err)
	}

	return nil
}
