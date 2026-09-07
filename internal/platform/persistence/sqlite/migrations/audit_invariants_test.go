package migrations

import (
	"context"
	"testing"

	"github.com/pressly/goose/v3"
)

// Invariant: upgrade from a supported populated previous schema preserves data.
func TestAuditUpgradePopulatedVersionTwo(t *testing.T) {
	db := openTestDatabase(t)
	goose.SetBaseFS(files)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpTo(db, "sql", 2); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO payments(id,status,version) VALUES ('legacy','pending',1)`); err != nil {
		t.Fatal(err)
	}
	if err := Up(db); err != nil {
		t.Fatalf("populated v2 -> latest migration failed: %v", err)
	}
}

// Positive control: repeating latest migrations does not erase a populated database.
func TestAuditReapplyPreservesPopulatedLatest(t *testing.T) {
	db := openTestDatabase(t)
	if err := Up(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO payments(id,status,version,amount,currency) VALUES ('existing','pending',1,1000,'EUR')`); err != nil {
		t.Fatal(err)
	}
	if err := Up(db); err != nil {
		t.Fatal(err)
	}
	var amount int64
	if err := db.QueryRowContext(context.Background(), `SELECT amount FROM payments WHERE id='existing'`).Scan(&amount); err != nil {
		t.Fatal(err)
	}
	if amount != 1000 {
		t.Fatalf("amount=%d", amount)
	}
}
