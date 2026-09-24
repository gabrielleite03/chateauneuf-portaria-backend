package database

import (
	"path/filepath"
	"testing"
)

func TestMigrationsCanRunAgainOnStartup(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "portaria.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	path := filepath.Join("..", "..", "migrations")
	if err := Migrate(db, path); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO residents (unit, owner, created_at, updated_at, authorized_recipients) VALUES ('11', 'Owner', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'Recipient')`); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db, path); err != nil {
		t.Fatalf("migrations failed on restart: %v", err)
	}
	var name string
	if err := db.QueryRow(`SELECT authorized_recipients FROM residents WHERE unit = '11'`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "Recipient" {
		t.Fatalf("resident changed on restart: %q", name)
	}
}
