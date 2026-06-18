package db

import (
	"context"
	"os"
	"sort"
	"testing"
	"time"
)

// TestEmbeddedMigrationsPresent verifies the migration files are embedded into
// the binary (deploy ships no source tree) and are name-ordered.
func TestEmbeddedMigrationsPresent(t *testing.T) {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) == 0 {
		t.Fatal("no embedded migrations found")
	}
	if !sort.StringsAreSorted(names) {
		// ReadDir already returns sorted names, but assert the invariant we rely on.
		t.Fatalf("embedded migrations not sorted: %v", names)
	}
	if names[0] != "0001_core.sql" {
		t.Fatalf("expected first migration 0001_core.sql, got %s", names[0])
	}
}

// TestMigrateAgainstDatabase runs the real migrations when HIGO_TEST_DATABASE_URL
// is set (e.g. a local/CI pgvector). It asserts idempotency. Skipped otherwise.
func TestMigrateAgainstDatabase(t *testing.T) {
	dsn := os.Getenv("HIGO_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set HIGO_TEST_DATABASE_URL to run migration integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := Migrate(ctx, dsn); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	applied, err := Migrate(ctx, dsn)
	if err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	if applied != 0 {
		t.Fatalf("expected 0 migrations on re-run, got %d", applied)
	}
}
