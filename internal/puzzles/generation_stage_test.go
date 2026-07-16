package puzzles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestGenerationStageUsesDisposableSQLiteModes(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	source := testSource("stage-modes", "test", "/stage-modes")
	importing := beginGenerationImport(t, catalog, source)
	staged, ok := importing.(*sqliteGenerationImport)
	if !ok {
		t.Fatalf("generation import type = %T, want *sqliteGenerationImport", importing)
	}
	defer func() {
		if err := importing.Abandon(context.Background()); err != nil {
			t.Errorf("abandon stage-mode import: %v", err)
		}
	}()

	var journalMode, lockingMode string
	var synchronous, busyTimeout, cacheSize, tempStore int
	if err := staged.stage.db.QueryRow(`PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if err := staged.stage.db.QueryRow(`PRAGMA synchronous`).Scan(&synchronous); err != nil {
		t.Fatal(err)
	}
	if err := staged.stage.db.QueryRow(`PRAGMA locking_mode`).Scan(&lockingMode); err != nil {
		t.Fatal(err)
	}
	if err := staged.stage.db.QueryRow(`PRAGMA busy_timeout`).Scan(&busyTimeout); err != nil {
		t.Fatal(err)
	}
	if err := staged.stage.db.QueryRow(`PRAGMA cache_size`).Scan(&cacheSize); err != nil {
		t.Fatal(err)
	}
	if err := staged.stage.db.QueryRow(`PRAGMA temp_store`).Scan(&tempStore); err != nil {
		t.Fatal(err)
	}
	if journalMode != "off" || synchronous != 0 || lockingMode != "exclusive" ||
		busyTimeout != 5_000 || cacheSize != -65_536 || tempStore != 1 {
		t.Fatalf(
			"stage modes journal=%q synchronous=%d locking=%q busy=%d cache=%d temp=%d, want off/0/exclusive/5000/-65536/1",
			journalMode,
			synchronous,
			lockingMode,
			busyTimeout,
			cacheSize,
			tempStore,
		)
	}
}

func TestGenerationWinnerUsesDisposableSQLiteModes(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	importing := beginGenerationImport(
		t,
		catalog,
		testSource("winner-modes", "test", "/winner-modes"),
	)
	defer func() {
		if err := importing.Abandon(context.Background()); err != nil {
			t.Errorf("abandon winner-mode import: %v", err)
		}
	}()
	staged := importing.(*sqliteGenerationImport)
	winner, err := createGenerationWinner(
		context.Background(),
		catalog.writeDB,
		staged.generationID,
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := winner.closeAndRemove(); err != nil {
			t.Errorf("remove winner-mode fixture: %v", err)
		}
	})

	var journalMode, lockingMode string
	var synchronous, busyTimeout, cacheSize, tempStore int
	for query, destination := range map[string]any{
		`PRAGMA journal_mode`: &journalMode,
		`PRAGMA synchronous`:  &synchronous,
		`PRAGMA locking_mode`: &lockingMode,
		`PRAGMA busy_timeout`: &busyTimeout,
		`PRAGMA cache_size`:   &cacheSize,
		`PRAGMA temp_store`:   &tempStore,
	} {
		if err := winner.db.QueryRow(query).Scan(destination); err != nil {
			t.Fatal(err)
		}
	}
	if journalMode != "off" || synchronous != 0 || lockingMode != "exclusive" ||
		busyTimeout != 5_000 || cacheSize != -65_536 || tempStore != 1 {
		t.Fatalf(
			"winner modes journal=%q synchronous=%d locking=%q busy=%d cache=%d temp=%d, want off/0/exclusive/5000/-65536/1",
			journalMode,
			synchronous,
			lockingMode,
			busyTimeout,
			cacheSize,
			tempStore,
		)
	}
}

func TestGenerationStageCoreConflictValidationAvoidsStagedRowProbePass(t *testing.T) {
	staged := generationAppendPlanFixture(t, "core-conflict-plan")
	winner, _, _, err := staged.compactToWinner(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := winner.closeAndRemove(); err != nil {
			t.Errorf("remove core-conflict winner fixture: %v", err)
		}
	})
	winnerDB, err := sql.Open("sqlite", winner.path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := winnerDB.Close(); err != nil {
			t.Errorf("close core-conflict winner fixture: %v", err)
		}
	})
	details := catalogQueryPlanDetails(
		t,
		winnerDB,
		generationWinnerCoreConflictQuery,
	)
	assertQueryPlanContains(t, details, "winner_core_conflicts")
	assertQueryPlanNotContains(t, details, "SCAN staged")
}

func TestGenerationWinnerBuildUsesOneWideSortWithoutStagedLookup(t *testing.T) {
	staged := generationAppendPlanFixture(t, "winner-wide-sort-plan")
	winner, err := createGenerationWinner(
		context.Background(),
		staged.catalog.writeDB,
		staged.generationID,
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := winner.closeAndRemove(); err != nil {
			t.Errorf("remove winner plan fixture: %v", err)
		}
	})
	if err := staged.stage.close(); err != nil {
		t.Fatal(err)
	}
	if _, err := winner.db.Exec(
		`ATTACH DATABASE ? AS `+generationAppendSchema,
		staged.stage.path,
	); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if winner.db == nil {
			return
		}
		if _, err := winner.db.Exec(`DETACH DATABASE ` + generationAppendSchema); err != nil {
			t.Errorf("detach append plan fixture: %v", err)
		}
	})

	details := catalogQueryPlanDetails(t, winner.db, generationWinnerBuildSQL)
	assertQueryPlanContains(t, details, "SCAN staged")
	assertQueryPlanNotContains(t, details, "SEARCH staged")
	assertQueryPlanNotContains(t, details, "USE TEMP B-TREE FOR GROUP BY")
	if sorts := countQueryPlanDetails(details, "USE TEMP B-TREE FOR ORDER BY"); sorts != 1 {
		t.Fatalf("winner build temp sorts = %d, want exactly 1; plan=%v", sorts, details)
	}
}

func TestGenerationWinnerCompactionCreatesFullWidthClusterAndConsumesAppend(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	source := testSource("winner-full-width", "test", "/winner-full-width")
	importing := beginGenerationImport(t, catalog, source)
	staged := importing.(*sqliteGenerationImport)
	defer func() {
		if err := importing.Abandon(context.Background()); err != nil {
			t.Errorf("abandon winner compaction import: %v", err)
		}
	}()

	first := testTrainingPuzzle(source, "winner-a", 1200, "fork")
	first.Occurrence.Ordinal = 9
	other := testTrainingPuzzle(source, "winner-b", 1300, "pin")
	other.Occurrence.Ordinal = 8
	last := testTrainingPuzzle(source, "winner-a", 1500, "mate")
	last.Occurrence.Ordinal = 1
	for _, puzzle := range []TrainingPuzzle{first, other, last} {
		if err := importing.Add(context.Background(), puzzle); err != nil {
			t.Fatal(err)
		}
	}
	if err := staged.flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	appendPath := staged.stage.path
	winner, stagedRows, winners, err := staged.compactToWinner(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := winner.closeAndRemove(); err != nil {
			t.Errorf("remove compacted winner: %v", err)
		}
	})
	if stagedRows != 3 || winners != 2 {
		t.Fatalf("compaction counts staged=%d winners=%d, want 3/2", stagedRows, winners)
	}
	if staged.stage != nil {
		t.Fatal("compaction retained append-stage ownership")
	}
	if _, err := os.Stat(appendPath); !os.IsNotExist(err) {
		t.Fatalf("append stage after compaction stat = %v, want not-exist", err)
	}
	if winner.db != nil {
		t.Fatal("compaction returned an open winner database")
	}
	if _, err := os.Stat(winner.path); err != nil {
		t.Fatalf("stat compacted winner: %v", err)
	}

	winnerDB, err := sql.Open("sqlite", winner.path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := winnerDB.Close(); err != nil {
			t.Errorf("close compacted winner fixture: %v", err)
		}
	})
	var tableSQL string
	if err := winnerDB.QueryRow(
		`SELECT sql FROM sqlite_schema WHERE type = 'table' AND name = 'winner_rows'`,
	).Scan(&tableSQL); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToUpper(tableSQL), "WITHOUT ROWID") {
		t.Fatalf("winner schema is not clustered WITHOUT ROWID: %s", tableSQL)
	}
	rows, err := winnerDB.Query(`SELECT
		fingerprint, rating, ordinal, themes_json, copies, core_conflict
		FROM winner_rows`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	type winnerRow struct {
		fingerprint  string
		rating       int
		ordinal      int64
		themesJSON   string
		copies       int64
		coreConflict int
	}
	var got []winnerRow
	for rows.Next() {
		var row winnerRow
		if err := rows.Scan(
			&row.fingerprint,
			&row.rating,
			&row.ordinal,
			&row.themesJSON,
			&row.copies,
			&row.coreConflict,
		); err != nil {
			t.Fatal(err)
		}
		got = append(got, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].fingerprint != "winner-a" || got[0].rating != 1500 ||
		got[0].ordinal != 1 || got[0].themesJSON != `["mate"]` || got[0].copies != 2 ||
		got[0].coreConflict != 0 || got[1].fingerprint != "winner-b" || got[1].copies != 1 {
		t.Fatalf("compacted winner rows = %+v", got)
	}
}

func countQueryPlanDetails(details []string, fragment string) int {
	count := 0
	for _, detail := range details {
		if strings.Contains(detail, fragment) {
			count++
		}
	}
	return count
}

func generationAppendPlanFixture(t *testing.T, name string) *sqliteGenerationImport {
	t.Helper()
	catalog, _ := openTestGenerationalCatalog(t)
	source := testSource(name, "test", "/"+name)
	importing := beginGenerationImport(t, catalog, source)
	t.Cleanup(func() {
		if err := importing.Abandon(context.Background()); err != nil {
			t.Errorf("abandon append plan fixture: %v", err)
		}
	})
	for index, fingerprint := range []string{name + "-b", name + "-a", name + "-a"} {
		puzzle := testTrainingPuzzle(source, fingerprint, 1200+index, "fork")
		puzzle.Occurrence.Ordinal = int64(index + 1)
		if err := importing.Add(context.Background(), puzzle); err != nil {
			t.Fatal(err)
		}
	}
	staged := importing.(*sqliteGenerationImport)
	if err := staged.flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	return staged
}

func TestGenerationWinnerMaterializationUsesSequentialSourceScans(t *testing.T) {
	staged := generationAppendPlanFixture(t, "winner-materialization-plan")
	attachGenerationWinnerPlanFixture(t, staged)
	tests := []struct {
		name      string
		statement string
		args      []any
		tempSorts int
	}{
		{name: "cores", statement: generationCoreMaterializationSQL},
		{
			name:      "occurrences",
			statement: generationOccurrenceMaterializationSQL,
			args:      []any{staged.generationID},
		},
		{
			name:      "ratings",
			statement: generationRatingMaterializationSQL,
			args: []any{
				staged.generationID,
				nullPuzzleRatingKey,
				nullPuzzleRatingKey,
			},
			tempSorts: 1,
		},
		{
			name:      "themes",
			statement: generationThemeMaterializationSQL,
			args:      []any{staged.generationID},
			tempSorts: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			details := catalogQueryPlanDetails(
				t,
				staged.catalog.writeDB,
				test.statement,
				test.args...,
			)
			assertQueryPlanContains(t, details, "SCAN winner")
			assertQueryPlanNotContains(t, details, "SCAN staged")
			assertQueryPlanNotContains(t, details, "SEARCH staged")
			if sorts := countQueryPlanDetails(details, "USE TEMP B-TREE FOR ORDER BY"); sorts != test.tempSorts {
				t.Fatalf(
					"%s materialization temp sorts = %d, want %d; plan=%v",
					test.name,
					sorts,
					test.tempSorts,
					details,
				)
			}
		})
	}
}

func TestMembershipMaterializationAuditsUseOneSortWithoutParentProbes(t *testing.T) {
	staged := generationAppendPlanFixture(t, "membership-audit-plan")
	attachGenerationWinnerPlanFixture(t, staged)
	tests := []struct {
		name      string
		statement string
		args      []any
		actual    string
	}{
		{
			name:      "ratings",
			statement: generationRatingMaterializationAuditSQL,
			args: []any{
				staged.generationID,
				nullPuzzleRatingKey,
				staged.generationID,
			},
			actual: "rated USING PRIMARY KEY (generation_id=?)",
		},
		{
			name:      "themes",
			statement: generationThemeMaterializationAuditSQL,
			args:      []any{staged.generationID, staged.generationID},
			actual:    "themed USING PRIMARY KEY (generation_id=?)",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			details := catalogQueryPlanDetails(
				t,
				staged.catalog.writeDB,
				test.statement,
				test.args...,
			)
			assertQueryPlanContains(t, details, "SCAN winner")
			assertQueryPlanContains(t, details, test.actual)
			assertQueryPlanNotContains(t, details, "SEARCH winner")
			assertQueryPlanNotContains(t, details, "SEARCH occurrence")
			if sorts := countQueryPlanDetails(details, "USE TEMP B-TREE FOR GROUP BY"); sorts != 1 {
				t.Fatalf("%s audit temp sorts = %d, want 1; plan=%v", test.name, sorts, details)
			}
		})
	}
}

func TestMembershipMaterializationSuspendsForeignKeysAndRestoresThem(t *testing.T) {
	staged := generationAppendPlanFixture(t, "membership-foreign-key-mode")
	winner, stagedRows, winners, err := staged.compactToWinner(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := winner.closeAndRemove(); err != nil {
			t.Errorf("remove membership foreign-key winner: %v", err)
		}
	})
	staged.report.Accepted = winners
	staged.report.Duplicates = stagedRows - winners

	if _, err := staged.catalog.writeDB.Exec(`
		CREATE TEMP TRIGGER require_rating_bulk_foreign_keys_off
		BEFORE INSERT ON occurrence_ratings
		WHEN (SELECT foreign_keys FROM pragma_foreign_keys) <> 0
		BEGIN
			SELECT RAISE(ABORT, 'rating bulk insert kept foreign keys enabled');
		END;
		CREATE TEMP TRIGGER require_theme_bulk_foreign_keys_off
		BEFORE INSERT ON occurrence_themes
		WHEN (SELECT foreign_keys FROM pragma_foreign_keys) <> 0
		BEGIN
			SELECT RAISE(ABORT, 'theme bulk insert kept foreign keys enabled');
		END;
	`); err != nil {
		t.Fatal(err)
	}

	if err := staged.materializeWinner(context.Background(), winner); err != nil {
		t.Fatal(err)
	}
	var foreignKeys int
	if err := staged.catalog.writeDB.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys after membership materialization = %d, want 1", foreignKeys)
	}
}

func TestMembershipMaterializationRestoresForeignKeysAfterBulkInsertFailure(t *testing.T) {
	staged := generationAppendPlanFixture(t, "membership-foreign-key-error")
	winner, stagedRows, winners, err := staged.compactToWinner(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := winner.closeAndRemove(); err != nil {
			t.Errorf("remove failing membership winner: %v", err)
		}
	})
	staged.report.Accepted = winners
	staged.report.Duplicates = stagedRows - winners

	if _, err := staged.catalog.writeDB.Exec(`
		CREATE TEMP TRIGGER fail_rating_bulk_with_foreign_keys_off
		BEFORE INSERT ON occurrence_ratings
		WHEN (SELECT foreign_keys FROM pragma_foreign_keys) = 0
		BEGIN
			SELECT RAISE(ABORT, 'forced rating bulk failure');
		END;
	`); err != nil {
		t.Fatal(err)
	}

	err = staged.materializeWinner(context.Background(), winner)
	if err == nil || !strings.Contains(err.Error(), "forced rating bulk failure") {
		t.Fatalf("materializeWinner() err = %v, want forced rating bulk failure", err)
	}
	var foreignKeys int
	if err := staged.catalog.writeDB.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys after failed membership materialization = %d, want 1", foreignKeys)
	}
}

func TestMembershipMaterializationAuditRejectsExtraRating(t *testing.T) {
	staged := generationAppendPlanFixture(t, "membership-foreign-key-check")
	winner, stagedRows, winners, err := staged.compactToWinner(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := winner.closeAndRemove(); err != nil {
			t.Errorf("remove checked membership winner: %v", err)
		}
	})
	staged.report.Accepted = winners
	staged.report.Duplicates = stagedRows - winners

	if _, err := staged.catalog.writeDB.Exec(`
		CREATE TEMP TRIGGER inject_orphan_during_theme_bulk
		AFTER INSERT ON occurrence_themes
		WHEN (SELECT foreign_keys FROM pragma_foreign_keys) = 0
		BEGIN
			INSERT OR IGNORE INTO occurrence_ratings(
				generation_id, rating_key, fingerprint
			) VALUES (NEW.generation_id, 0, 'orphan-fingerprint');
		END;
	`); err != nil {
		t.Fatal(err)
	}

	err = staged.materializeWinner(context.Background(), winner)
	if err == nil || !strings.Contains(err.Error(), "rating materialization audit") {
		t.Fatalf("materializeWinner() err = %v, want rating materialization audit failure", err)
	}
	var foreignKeys int
	if err := staged.catalog.writeDB.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys after failed exact check = %d, want 1", foreignKeys)
	}
}

func TestMembershipMaterializationAuditRejectsReplacedRating(t *testing.T) {
	staged := generationAppendPlanFixture(t, "membership-rating-audit")
	winner, stagedRows, winners, err := staged.compactToWinner(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := winner.closeAndRemove(); err != nil {
			t.Errorf("remove rating-audit winner: %v", err)
		}
	})
	staged.report.Accepted = winners
	staged.report.Duplicates = stagedRows - winners

	if _, err := staged.catalog.writeDB.Exec(`
		CREATE TEMP TRIGGER replace_rating_during_bulk
		BEFORE INSERT ON occurrence_ratings
		WHEN (SELECT foreign_keys FROM pragma_foreign_keys) = 0
		BEGIN
			INSERT INTO occurrence_ratings(
				generation_id, rating_key, fingerprint
			) VALUES (NEW.generation_id, NEW.rating_key + 1, NEW.fingerprint);
			SELECT RAISE(IGNORE);
		END;
	`); err != nil {
		t.Fatal(err)
	}

	err = staged.materializeWinner(context.Background(), winner)
	if err == nil || !strings.Contains(err.Error(), "rating materialization audit") {
		t.Fatalf("materializeWinner() err = %v, want rating materialization audit failure", err)
	}
}

func TestMembershipMaterializationAuditRejectsMissingTheme(t *testing.T) {
	staged := generationAppendPlanFixture(t, "membership-theme-audit")
	winner, stagedRows, winners, err := staged.compactToWinner(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := winner.closeAndRemove(); err != nil {
			t.Errorf("remove theme-audit winner: %v", err)
		}
	})
	staged.report.Accepted = winners
	staged.report.Duplicates = stagedRows - winners

	if _, err := staged.catalog.writeDB.Exec(`
		CREATE TEMP TRIGGER omit_theme_during_bulk
		BEFORE INSERT ON occurrence_themes
		WHEN (SELECT foreign_keys FROM pragma_foreign_keys) = 0
		BEGIN
			SELECT RAISE(IGNORE);
		END;
	`); err != nil {
		t.Fatal(err)
	}

	err = staged.materializeWinner(context.Background(), winner)
	if err == nil || !strings.Contains(err.Error(), "theme materialization audit") {
		t.Fatalf("materializeWinner() err = %v, want theme materialization audit failure", err)
	}
}

func TestExistingCoreConflictValidationStartsFromCanonicalCores(t *testing.T) {
	staged := generationAppendPlanFixture(t, "existing-core-plan")
	attachGenerationWinnerPlanFixture(t, staged)
	details := catalogQueryPlanDetails(
		t,
		staged.catalog.writeDB,
		generationExistingCoreConflictQuery,
	)
	assertQueryPlanContains(t, details, "SCAN core")
	assertQueryPlanNotContains(t, details, "SCAN winner")
}

func attachGenerationWinnerPlanFixture(t *testing.T, staged *sqliteGenerationImport) {
	t.Helper()
	winner, _, _, err := staged.compactToWinner(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := staged.catalog.writeDB.Exec(
		`ATTACH DATABASE ? AS `+generationWinnerSchema,
		winner.path,
	); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := staged.catalog.writeDB.Exec(
			`DETACH DATABASE ` + generationWinnerSchema,
		); err != nil {
			t.Errorf("detach generation-winner plan fixture: %v", err)
		}
		if err := winner.closeAndRemove(); err != nil {
			t.Errorf("remove generation-winner plan fixture: %v", err)
		}
	})
}

func TestFailedSealRemovesStageAndPreservesHead(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	source := testSource("failed-seal-stage", "test", "/failed-seal-active")
	sealAndActivate(
		t,
		beginGenerationImport(t, catalog, source),
		testTrainingPuzzle(source, "failed-seal-active", 1200, "fork"),
	)
	replacementSource := source
	replacementSource.Path = "/failed-seal-replacement"
	replacement := beginGenerationImport(t, catalog, replacementSource)
	staged := replacement.(*sqliteGenerationImport)
	stagePath := staged.stage.path
	winnerPath := generationWinnerPathForStage(stagePath)
	first := testTrainingPuzzle(source, "failed-seal-conflict", 1400, "fork")
	mismatch := testTrainingPuzzle(source, "failed-seal-conflict", 1500, "pin")
	mismatch.Occurrence.Ordinal = 2
	mismatch.Core.DisplayedFEN = "8/8/8/8/8/8/5Q2/7k w - - 0 1"
	for _, puzzle := range []TrainingPuzzle{first, mismatch} {
		if err := replacement.Add(context.Background(), puzzle); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := replacement.Seal(context.Background(), "failed-seal"); !errors.Is(err, ErrCatalogCorrupt) {
		t.Fatalf("Seal(conflicting core) err = %v, want ErrCatalogCorrupt", err)
	}
	if _, err := os.Stat(stagePath); !os.IsNotExist(err) {
		t.Fatalf("failed-seal stage stat = %v, want not-exist", err)
	}
	if _, err := os.Stat(winnerPath); !os.IsNotExist(err) {
		t.Fatalf("failed-seal winner stat = %v, want not-exist", err)
	}
	if _, err := catalog.Get(context.Background(), PuzzleKey{
		Fingerprint: "failed-seal-active",
		SourceID:    source.ID,
	}); err != nil {
		t.Fatalf("active head after failed seal: %v", err)
	}
	if err := replacement.Add(
		context.Background(),
		testTrainingPuzzle(source, "failed-seal-after-error", 1600),
	); err == nil {
		t.Fatal("Add() accepted a row after finalization consumed the stage")
	}
	if _, err := replacement.Seal(context.Background(), "failed-seal-retry"); err == nil {
		t.Fatal("Seal() retried after finalization consumed the stage")
	}
}

func TestSealCheckpointsBulkPhasesAndRestoresAutoCheckpoint(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	var busy, logPages, checkpointed int
	if err := store.Writer.QueryRow(`PRAGMA wal_checkpoint(TRUNCATE)`).Scan(
		&busy,
		&logPages,
		&checkpointed,
	); err != nil {
		t.Fatal(err)
	}
	source := testSource("checkpointed-seal", "test", "/checkpointed-seal")
	importing := beginGenerationImport(t, catalog, source)
	for index := range generationImportBatchSize {
		puzzle := testTrainingPuzzle(
			source,
			fmt.Sprintf("checkpointed-%04d", generationImportBatchSize-index),
			1200+(index%200),
			"fork",
			"pin",
		)
		puzzle.Occurrence.Ordinal = int64(index + 1)
		if err := importing.Add(context.Background(), puzzle); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := importing.Seal(context.Background(), "checkpointed-seal"); err != nil {
		t.Fatal(err)
	}
	rows, err := store.Reader.Query(`SELECT fingerprint FROM puzzle_cores ORDER BY rowid`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	inserted := 0
	for rows.Next() {
		inserted++
		var fingerprint string
		if err := rows.Scan(&fingerprint); err != nil {
			t.Fatal(err)
		}
		want := fmt.Sprintf("checkpointed-%04d", inserted)
		if fingerprint != want {
			t.Fatalf("core rowid %d fingerprint = %q, want %q", inserted, fingerprint, want)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if inserted != generationImportBatchSize {
		t.Fatalf("materialized core rows = %d, want %d", inserted, generationImportBatchSize)
	}

	var autoCheckpoint int
	if err := store.Writer.QueryRow(`PRAGMA wal_autocheckpoint`).Scan(&autoCheckpoint); err != nil {
		t.Fatal(err)
	}
	if autoCheckpoint != 16_384 {
		t.Fatalf("wal_autocheckpoint after seal = %d, want 16384", autoCheckpoint)
	}
	if err := store.Writer.QueryRow(`PRAGMA wal_checkpoint(PASSIVE)`).Scan(
		&busy,
		&logPages,
		&checkpointed,
	); err != nil {
		t.Fatal(err)
	}
	if busy != 0 || logPages >= 64 {
		t.Fatalf(
			"post-seal WAL busy=%d log_pages=%d checkpointed=%d, want busy=0 log_pages<64",
			busy,
			logPages,
			checkpointed,
		)
	}
}

func TestSealRestoresAutoCheckpointWhenBulkPhaseFails(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	if _, err := store.Writer.Exec(`CREATE TRIGGER fail_occurrence_materialization
		BEFORE INSERT ON puzzle_occurrences
		BEGIN
		  SELECT RAISE(ABORT, 'injected occurrence materialization failure');
		END`); err != nil {
		t.Fatal(err)
	}
	source := testSource("failed-checkpointed-seal", "test", "/failed-checkpointed-seal")
	importing := beginGenerationImport(t, catalog, source)
	stagePath := importing.(*sqliteGenerationImport).stage.path
	winnerPath := generationWinnerPathForStage(stagePath)
	if err := importing.Add(
		context.Background(),
		testTrainingPuzzle(source, "failed-checkpointed-seal", 1200, "fork"),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := importing.Seal(context.Background(), "failed-checkpointed-seal"); err == nil {
		t.Fatal("Seal() succeeded despite injected occurrence materialization failure")
	}
	var autoCheckpoint int
	if err := store.Writer.QueryRow(`PRAGMA wal_autocheckpoint`).Scan(&autoCheckpoint); err != nil {
		t.Fatal(err)
	}
	if autoCheckpoint != 16_384 {
		t.Fatalf("wal_autocheckpoint after failed seal = %d, want 16384", autoCheckpoint)
	}
	if _, err := os.Stat(stagePath); !os.IsNotExist(err) {
		t.Fatalf("failed bulk stage stat = %v, want not-exist", err)
	}
	if _, err := os.Stat(winnerPath); !os.IsNotExist(err) {
		t.Fatalf("failed bulk winner stat = %v, want not-exist", err)
	}
}

func TestMaterializationRestoresAutoCheckpointWhenDisableVerificationFails(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	importing := beginGenerationImport(
		t,
		catalog,
		testSource("failed-checkpoint-verification", "test", "/failed-checkpoint-verification"),
	)
	defer func() {
		if err := importing.Abandon(context.Background()); err != nil {
			t.Errorf("abandon checkpoint-verification fixture: %v", err)
		}
	}()
	injected := errors.New("injected post-set verification failure")
	setThenFail := func(ctx context.Context, db *sql.DB, pages int) error {
		if pages != 0 {
			return setPuzzleWriterAutoCheckpoint(ctx, db, pages)
		}
		if _, err := db.ExecContext(ctx, `PRAGMA wal_autocheckpoint=0`); err != nil {
			return err
		}
		return injected
	}
	err := importing.(*sqliteGenerationImport).runMaterializationPhaseWithAutoCheckpoint(
		context.Background(),
		"injected checkpoint verification",
		`SELECT 1`,
		setThenFail,
	)
	if !errors.Is(err, injected) {
		t.Fatalf("materialization error = %v, want injected verification failure", err)
	}
	var autoCheckpoint int
	if err := store.Writer.QueryRow(`PRAGMA wal_autocheckpoint`).Scan(&autoCheckpoint); err != nil {
		t.Fatal(err)
	}
	if autoCheckpoint != 16_384 {
		t.Fatalf("wal_autocheckpoint after disable verification failure = %d, want 16384", autoCheckpoint)
	}
}

func TestSealIndexesNullRatingsWithSentinel(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testSource("null-rating-stage", "test", "/null-rating-stage")
	puzzle := testTrainingPuzzle(source, "null-rating-stage", 1200, "fork")
	puzzle.Occurrence.Rating = nil
	importing := beginGenerationImport(t, catalog, source)
	if err := importing.Add(context.Background(), puzzle); err != nil {
		t.Fatal(err)
	}
	if _, err := importing.Seal(context.Background(), "null-rating-stage"); err != nil {
		t.Fatal(err)
	}
	generationID := generationIDForPath(t, store.Reader, source.Path)
	var occurrenceRatingIsNull bool
	var ratingKey int64
	if err := store.Reader.QueryRow(`SELECT
		  occurrence.rating IS NULL,
		  rated.rating_key
		FROM puzzle_occurrences AS occurrence
		JOIN occurrence_ratings AS rated
		  ON rated.generation_id = occurrence.generation_id
		 AND rated.fingerprint = occurrence.fingerprint
		WHERE occurrence.generation_id = ?
		  AND occurrence.fingerprint = ?`, generationID, puzzle.Core.Fingerprint).Scan(
		&occurrenceRatingIsNull,
		&ratingKey,
	); err != nil {
		t.Fatal(err)
	}
	if !occurrenceRatingIsNull || ratingKey != nullPuzzleRatingKey {
		t.Fatalf(
			"null rating materialization: occurrence_null=%t rating_key=%d, want true/%d",
			occurrenceRatingIsNull,
			ratingKey,
			nullPuzzleRatingKey,
		)
	}
}

func TestAbandonRemovesStageWhenContextIsCanceled(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	source := testSource("cancelled-abandon-stage", "test", "/cancelled-abandon-stage")
	importing := beginGenerationImport(t, catalog, source)
	stagePath := importing.(*sqliteGenerationImport).stage.path
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := importing.Abandon(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("Abandon(canceled) err = %v, want context.Canceled", err)
	}
	if _, err := os.Stat(stagePath); !os.IsNotExist(err) {
		t.Fatalf("canceled-abandon stage stat = %v, want not-exist", err)
	}
}

func generationWinnerPathForStage(stagePath string) string {
	return strings.Replace(stagePath, ".stage-", ".winner-", 1)
}
