package storage

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestCurrentPuzzlePhysicalSchemaRejectsTampering(t *testing.T) {
	mutations := []struct {
		name   string
		mutate func(*sql.DB) error
	}{
		{
			name:   "rowid occurrence ratings",
			mutate: recreateOccurrenceRatingsAsRowID,
		},
		{
			name: "changed occurrence rating primary key",
			mutate: func(db *sql.DB) error {
				return recreateOccurrenceRatings(db, "PRIMARY KEY (generation_id, fingerprint, rating_key)", "ON DELETE CASCADE")
			},
		},
		{
			name: "descending occurrence rating key",
			mutate: func(db *sql.DB) error {
				return recreateOccurrenceRatings(db, "PRIMARY KEY (generation_id DESC, rating_key, fingerprint)", "ON DELETE CASCADE")
			},
		},
		{
			name: "nocase occurrence rating key",
			mutate: func(db *sql.DB) error {
				return recreateOccurrenceRatings(db, "PRIMARY KEY (generation_id COLLATE NOCASE, rating_key, fingerprint)", "ON DELETE CASCADE")
			},
		},
		{
			name: "extra explicit index",
			mutate: func(db *sql.DB) error {
				_, err := db.Exec(`CREATE INDEX unexpected_occurrence_index
					ON puzzle_occurrences(generation_id, fingerprint)`)
				return err
			},
		},
		{
			name: "missing occurrence generation index",
			mutate: func(db *sql.DB) error {
				_, err := db.Exec(`DROP INDEX idx_occurrences_generation`)
				return err
			},
		},
		{
			name:   "rowid puzzle occurrences",
			mutate: recreatePuzzleOccurrencesAsRowID,
		},
		{
			name:   "occurrence themes without cascade",
			mutate: recreateOccurrenceThemesWithoutCascade,
		},
		{
			name: "occurrence ratings without cascade",
			mutate: func(db *sql.DB) error {
				return recreateOccurrenceRatings(db, "PRIMARY KEY (generation_id, rating_key, fingerprint)", "")
			},
		},
		{
			name:   "changed occurrence theme primary key",
			mutate: recreateOccurrenceThemesWithFingerprintFirst,
		},
		{
			name: "missing source generation composite unique",
			mutate: func(db *sql.DB) error {
				return recreateSourceGenerationsWithConstraints(db, "")
			},
		},
		{
			name: "changed source generation composite unique",
			mutate: func(db *sql.DB) error {
				return recreateSourceGenerationsWithConstraints(
					db,
					", UNIQUE (generation_id, source_id)",
				)
			},
		},
		{
			name: "extra source generation unique",
			mutate: func(db *sql.DB) error {
				return recreateSourceGenerationsWithConstraints(
					db,
					", UNIQUE (source_id, generation_id), UNIQUE (source_id, status)",
				)
			},
		},
	}

	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "puzzles.sqlite")
			store, err := OpenPuzzleStore(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}

			db := openPuzzleValidationFixture(t, path)
			if err := mutation.mutate(db); err != nil {
				t.Fatalf("mutate current schema: %v", err)
			}

			if err := validatePuzzleSchema(db); err == nil {
				t.Error("post-migrate validation accepted the tampered current schema")
			}
			state, err := ProbePuzzleStore(path)
			if err == nil {
				t.Error("ProbePuzzleStore accepted the tampered current schema")
			}
			if !state.Exists || state.Legacy || state.Format != CurrentPuzzleSchemaVersion {
				t.Fatalf("ProbePuzzleStore state = %+v, want rejected existing current v%d", state, CurrentPuzzleSchemaVersion)
			}
		})
	}
}

func openPuzzleValidationFixture(t *testing.T, path string) *sql.DB {
	t.Helper()
	dsn, err := puzzleStoreDSN(path, false, false)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close physical-schema fixture: %v", err)
		}
	})
	return db
}

func recreatePuzzleOccurrencesAsRowID(db *sql.DB) error {
	_, err := db.Exec(`PRAGMA foreign_keys=OFF;
		DROP TABLE occurrence_ratings;
		DROP TABLE occurrence_themes;
		DROP TABLE puzzle_occurrences;
		CREATE TABLE puzzle_occurrences (
		  generation_id TEXT NOT NULL
		    REFERENCES source_generations(generation_id) ON DELETE CASCADE,
		  fingerprint TEXT NOT NULL REFERENCES puzzle_cores(fingerprint),
		  external_id TEXT,
		  source_fen TEXT,
		  prelude_uci TEXT,
		  rating INTEGER,
		  popularity INTEGER,
		  play_count INTEGER,
		  source_url TEXT,
		  attribution TEXT,
		  metadata_json TEXT NOT NULL DEFAULT '{}',
		  themes_json TEXT NOT NULL DEFAULT '[]',
		  ordinal INTEGER NOT NULL CHECK (ordinal > 0),
		  PRIMARY KEY (fingerprint, generation_id)
		);
		CREATE TABLE occurrence_themes (
		  generation_id TEXT NOT NULL,
		  fingerprint TEXT NOT NULL,
		  theme TEXT NOT NULL,
		  PRIMARY KEY (generation_id, theme, fingerprint),
		  FOREIGN KEY (fingerprint, generation_id)
		    REFERENCES puzzle_occurrences(fingerprint, generation_id) ON DELETE CASCADE
		) WITHOUT ROWID;
		CREATE TABLE occurrence_ratings (
		  generation_id TEXT NOT NULL,
		  rating_key INTEGER NOT NULL,
		  fingerprint TEXT NOT NULL,
		  PRIMARY KEY (generation_id, rating_key, fingerprint),
		  FOREIGN KEY (fingerprint, generation_id)
		    REFERENCES puzzle_occurrences(fingerprint, generation_id) ON DELETE CASCADE
		) WITHOUT ROWID;
		PRAGMA foreign_keys=ON;`)
	return err
}

func recreateOccurrenceRatingsAsRowID(db *sql.DB) error {
	_, err := db.Exec(`PRAGMA foreign_keys=OFF;
		DROP TABLE occurrence_ratings;
		CREATE TABLE occurrence_ratings (
		  generation_id TEXT NOT NULL,
		  rating_key INTEGER NOT NULL,
		  fingerprint TEXT NOT NULL,
		  PRIMARY KEY (generation_id, rating_key, fingerprint),
		  FOREIGN KEY (fingerprint, generation_id)
		    REFERENCES puzzle_occurrences(fingerprint, generation_id) ON DELETE CASCADE
		);
		PRAGMA foreign_keys=ON;`)
	return err
}

func recreateOccurrenceRatings(db *sql.DB, primaryKey, deleteAction string) error {
	_, err := db.Exec(`PRAGMA foreign_keys=OFF;
		DROP TABLE occurrence_ratings;
		CREATE TABLE occurrence_ratings (
		  generation_id TEXT NOT NULL,
		  rating_key INTEGER NOT NULL,
		  fingerprint TEXT NOT NULL,
		  ` + primaryKey + `,
		  FOREIGN KEY (fingerprint, generation_id)
		    REFERENCES puzzle_occurrences(fingerprint, generation_id) ` + deleteAction + `
		) WITHOUT ROWID;
		PRAGMA foreign_keys=ON;`)
	return err
}

func recreateOccurrenceThemesWithoutCascade(db *sql.DB) error {
	_, err := db.Exec(`PRAGMA foreign_keys=OFF;
		DROP TABLE occurrence_themes;
		CREATE TABLE occurrence_themes (
		  generation_id TEXT NOT NULL,
		  fingerprint TEXT NOT NULL,
		  theme TEXT NOT NULL,
		  PRIMARY KEY (generation_id, theme, fingerprint),
		  FOREIGN KEY (fingerprint, generation_id)
		    REFERENCES puzzle_occurrences(fingerprint, generation_id)
		) WITHOUT ROWID;
		PRAGMA foreign_keys=ON;`)
	return err
}

func recreateOccurrenceThemesWithFingerprintFirst(db *sql.DB) error {
	_, err := db.Exec(`PRAGMA foreign_keys=OFF;
		DROP TABLE occurrence_themes;
		CREATE TABLE occurrence_themes (
		  generation_id TEXT NOT NULL,
		  fingerprint TEXT NOT NULL,
		  theme TEXT NOT NULL,
		  PRIMARY KEY (generation_id, fingerprint, theme),
		  FOREIGN KEY (fingerprint, generation_id)
		    REFERENCES puzzle_occurrences(fingerprint, generation_id) ON DELETE CASCADE
		) WITHOUT ROWID;
		PRAGMA foreign_keys=ON;`)
	return err
}

func recreateSourceGenerationsWithConstraints(db *sql.DB, constraints string) error {
	_, err := db.Exec(`PRAGMA foreign_keys=OFF;
		DROP INDEX idx_generations_cleanup;
		DROP TABLE source_generations;
		CREATE TABLE source_generations (
		  generation_id TEXT PRIMARY KEY,
		  source_id TEXT NOT NULL REFERENCES sources(source_id),
		  status TEXT NOT NULL CHECK(status IN ('building', 'sealed', 'abandoned')),
		  source_path TEXT NOT NULL,
		  checksum TEXT,
		  started_at INTEGER NOT NULL CHECK (started_at > 0),
		  sealed_at INTEGER CHECK (sealed_at IS NULL OR sealed_at > 0)` + constraints + `
		);
		CREATE INDEX idx_generations_cleanup
		  ON source_generations(status, generation_id);
		PRAGMA foreign_keys=ON;`)
	return err
}
