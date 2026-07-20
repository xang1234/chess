package storage

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestMigrateCreatesEachSchemaAndIsIdempotent(t *testing.T) {
	tests := []struct {
		schema     string
		table      string
		migrations int
	}{
		{schema: "puzzles", table: "puzzle_cores", migrations: 1},
		{schema: "user", table: "profile", migrations: 6},
		{schema: "library", table: "library_metadata", migrations: 1},
		{schema: "courses", table: "course_generations", migrations: 2},
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

func TestUserMigration005CreatesOpeningLearningState(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "user.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(db, "user"); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{
		"opening_preferences",
		"opening_course_journeys",
		"opening_sessions",
		"opening_lesson_progress",
		"opening_attempts",
		"opening_prompt_progress",
		"opening_review_state",
		"idx_opening_sessions_resume",
		"idx_opening_sessions_single_resumable",
		"idx_opening_reviews_due",
		"idx_opening_attempts_prompt",
	} {
		var found string
		if err := db.QueryRow(
			`SELECT name FROM sqlite_master WHERE name = ? AND type IN ('table','index')`,
			name,
		).Scan(&found); err != nil {
			t.Fatalf("schema object %q: %v", name, err)
		}
	}
}

func TestUserMigration006MakesOpeningPreferenceDepthCanonical(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "user.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	for version, name := range []string{"001.sql", "002.sql", "003.sql", "004.sql"} {
		body, err := os.ReadFile(filepath.Join("migrations", "user", name))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(body)); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
		if _, err := db.Exec(`INSERT INTO schema_migrations(version) VALUES (?)`, version+1); err != nil {
			t.Fatal(err)
		}
	}

	const createdAt = int64(1_753_003_200)
	const updatedAt = int64(1_753_006_800)
	if _, err := db.Exec(
		`INSERT INTO opening_preferences(course_id, depth, updated_at) VALUES
		 ('italian-white', 'quick', ?),
		 ('preference-only', 'reference', ?)`,
		updatedAt, updatedAt,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO opening_sessions(
		   session_id, course_id, generation_id, lesson_id, mode, status, depth,
		   step_index, state_json, created_at, updated_at
		 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"session-1", "italian-white", "generation-1", "giuoco-plan", "lesson", "active",
		"standard", 2, `{"restart":{"stepIndex":2}}`, createdAt, updatedAt,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO opening_lesson_progress(
		   course_id, lesson_id, completed_step_ids_json, completed_steps,
		   total_steps, completed_at, updated_at
		 ) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"italian-white", "foundations", `["concept","decision"]`, 2, 2, updatedAt, updatedAt,
	); err != nil {
		t.Fatal(err)
	}

	if err := Migrate(db, "user"); err != nil {
		t.Fatal(err)
	}

	var activityIndex int
	var stateJSON string
	if err := db.QueryRow(
		`SELECT activity_index, state_json FROM opening_sessions WHERE session_id = ?`,
		"session-1",
	).Scan(&activityIndex, &stateJSON); err != nil {
		t.Fatal(err)
	}
	if activityIndex != 2 || !strings.Contains(stateJSON, `"activityIndex":2`) || strings.Contains(stateJSON, `"stepIndex":`) {
		t.Fatalf("migrated session activity_index=%d state_json=%s", activityIndex, stateJSON)
	}

	var completedIDs string
	var completedActivities, totalActivities int
	var completedAt int64
	if err := db.QueryRow(
		`SELECT completed_activity_ids_json, completed_activities, total_activities, completed_at
		 FROM opening_lesson_progress WHERE course_id = ? AND lesson_id = ?`,
		"italian-white", "foundations",
	).Scan(&completedIDs, &completedActivities, &totalActivities, &completedAt); err != nil {
		t.Fatal(err)
	}
	if completedIDs != `["concept","decision"]` || completedActivities != 2 || totalActivities != 2 || completedAt != updatedAt {
		t.Fatalf(
			"migrated progress ids=%s completed=%d total=%d completed_at=%d",
			completedIDs, completedActivities, totalActivities, completedAt,
		)
	}

	var lessonID, pathJSON string
	var journeyCreatedAt, journeyUpdatedAt int64
	if err := db.QueryRow(
		`SELECT current_lesson_id, path_lesson_ids_json, created_at, updated_at
		 FROM opening_course_journeys WHERE course_id = ?`,
		"italian-white",
	).Scan(
		&lessonID, &pathJSON, &journeyCreatedAt, &journeyUpdatedAt,
	); err != nil {
		t.Fatal(err)
	}
	if lessonID != "giuoco-plan" || pathJSON != `["giuoco-plan"]` ||
		journeyCreatedAt != createdAt || journeyUpdatedAt != updatedAt {
		t.Fatalf(
			"migrated journey lesson=%s path=%s created=%d updated=%d",
			lessonID, pathJSON, journeyCreatedAt, journeyUpdatedAt,
		)
	}

	var migratedDepth string
	if err := db.QueryRow(
		`SELECT depth FROM opening_preferences WHERE course_id = ?`,
		"italian-white",
	).Scan(&migratedDepth); err != nil {
		t.Fatal(err)
	}
	if migratedDepth != "standard" {
		t.Fatalf("canonical preference depth=%s, want standard", migratedDepth)
	}

	rows, err := db.Query(`PRAGMA table_info(opening_course_journeys)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns := []string{}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, dataType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatal(err)
		}
		columns = append(columns, name)
	}
	wantColumns := []string{"course_id", "current_lesson_id", "path_lesson_ids_json", "created_at", "updated_at"}
	if !reflect.DeepEqual(columns, wantColumns) {
		t.Fatalf("journey columns=%v, want %v", columns, wantColumns)
	}
}

func TestUserMigration003AddsNullableSnapshotColumns(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "user.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(db, "user"); err != nil {
		t.Fatal(err)
	}

	want := map[string]map[string]string{
		"session_items": {
			"source_kind":          "TEXT",
			"rating_snapshot":      "INTEGER",
			"themes_json":          "TEXT",
			"source_fen_snapshot":  "TEXT",
			"prelude_uci_snapshot": "TEXT",
		},
		"attempts": {
			"source_kind":     "TEXT",
			"rating_snapshot": "INTEGER",
			"themes_json":     "TEXT",
		},
		"review_state": {
			"preferred_source_id": "TEXT",
		},
	}

	for table, columns := range want {
		rows, err := db.Query(`PRAGMA table_info("` + table + `")`)
		if err != nil {
			t.Fatal(err)
		}
		found := make(map[string]bool, len(columns))
		for rows.Next() {
			var cid, notNull, primaryKey int
			var name, columnType string
			var defaultValue any
			if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			wantType, ok := columns[name]
			if !ok {
				continue
			}
			found[name] = true
			if columnType != wantType || notNull != 0 || defaultValue != nil || primaryKey != 0 {
				rows.Close()
				t.Fatalf(
					"%s.%s signature = type %q notnull %d default %v pk %d, want %q nullable without default",
					table,
					name,
					columnType,
					notNull,
					defaultValue,
					primaryKey,
					wantType,
				)
			}
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		for name := range columns {
			if !found[name] {
				t.Errorf("%s.%s is missing", table, name)
			}
		}
	}
}

func TestCheckDurableIntegrityRejectsCorruptionWithoutCreatingStores(t *testing.T) {
	paths := PathsAt(filepath.Join(t.TempDir(), "data"))
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.UserDB, []byte("not a sqlite database"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := CheckDurableIntegrity(paths)
	var integrityErr *IntegrityError
	if !errors.As(err, &integrityErr) || integrityErr.Path != paths.UserDB || integrityErr.Detail == "" {
		t.Fatalf("CheckDurableIntegrity() err=%v", err)
	}
	for _, path := range []string{paths.PuzzlesDB, paths.LibraryDB} {
		if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("integrity check created %s: %v", path, statErr)
		}
	}
}

func TestCheckDurableIntegrityAcceptsValidDatabase(t *testing.T) {
	paths := PathsAt(filepath.Join(t.TempDir(), "data"))
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	db, err := Open(paths.UserDB)
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db, "user"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if err := CheckDurableIntegrity(paths); err != nil {
		t.Fatal(err)
	}
}
