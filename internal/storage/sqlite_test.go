package storage

import (
	"path/filepath"
	"testing"
)

func TestMigrateCreatesEachSchemaAndIsIdempotent(t *testing.T) {
	tests := []struct {
		schema     string
		table      string
		migrations int
	}{
		{schema: "puzzles", table: "puzzles", migrations: 1},
		{schema: "user", table: "profile", migrations: 2},
		{schema: "library", table: "library_metadata", migrations: 1},
	}

	for _, tt := range tests {
		t.Run(tt.schema, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tt.schema+".sqlite")
			db, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := Migrate(db, tt.schema); err != nil {
				t.Fatal(err)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}

			db, err = Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := Migrate(db, tt.schema); err != nil {
				t.Fatal(err)
			}

			var tableName string
			err = db.QueryRow(
				`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`,
				tt.table,
			).Scan(&tableName)
			if err != nil || tableName != tt.table {
				t.Fatalf("table=%q err=%v", tableName, err)
			}

			var migrations int
			if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&migrations); err != nil {
				t.Fatal(err)
			}
			if migrations != tt.migrations {
				t.Fatalf("migrations=%d, want %d", migrations, tt.migrations)
			}
		})
	}
}
