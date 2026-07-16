//go:build performance

package puzzles

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"chess-trainer/internal/storage"
)

const syntheticPerformanceRows = 100_000

func TestGenerationActivationCompletesWithinFiveSeconds(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	ctx := context.Background()
	source := testSource("activation-performance", "synthetic", "/activation-prior")
	prior, priorGenerationID := seedSyntheticSealedGeneration(
		t,
		catalog,
		store,
		source,
		"activation-prior",
		1,
	)
	if err := prior.Activate(ctx); err != nil {
		t.Fatal(err)
	}

	replacementSource := source
	replacementSource.Path = "/activation-replacement"
	replacement, replacementGenerationID := seedSyntheticSealedGeneration(
		t,
		catalog,
		store,
		replacementSource,
		"activation-replacement",
		syntheticPerformanceRows,
	)
	assertSourceHead(t, store, source.ID, priorGenerationID)
	assertGenerationOccurrenceCount(t, store, priorGenerationID, 1)
	assertGenerationOccurrenceCount(t, store, replacementGenerationID, syntheticPerformanceRows)

	started := time.Now()
	if err := replacement.Activate(ctx); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(started)
	if elapsed >= 5*time.Second {
		t.Fatalf("generation activation took %s, want less than 5s", elapsed)
	}

	assertSourceHead(t, store, source.ID, replacementGenerationID)
	assertGenerationOccurrenceCount(t, store, priorGenerationID, 1)
	assertGenerationOccurrenceCount(t, store, replacementGenerationID, syntheticPerformanceRows)
	var sourceHeads int
	if err := store.Reader.QueryRow(
		`SELECT COUNT(*) FROM source_heads WHERE source_id = ?`,
		source.ID,
	).Scan(&sourceHeads); err != nil {
		t.Fatal(err)
	}
	if sourceHeads != 1 {
		t.Fatalf("source head rows = %d, want exactly 1", sourceHeads)
	}
	t.Logf("activation_elapsed=%s sealed_occurrences=%d source_head_rows=%d", elapsed, syntheticPerformanceRows, sourceHeads)
}

func TestActiveCandidateQueryCompletesWithin250Milliseconds(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	ctx := context.Background()
	source := testSource("candidate-performance", "synthetic", "/candidate-active")
	active, _ := seedSyntheticSealedGeneration(
		t,
		catalog,
		store,
		source,
		"candidate-active",
		syntheticPerformanceRows,
	)
	if err := active.Activate(ctx); err != nil {
		t.Fatal(err)
	}
	warmPuzzleReadPool(t, store.Reader)
	warm, err := catalog.RatedCandidates(ctx, 1400, 1600, nil, 25)
	if err != nil {
		t.Fatal(err)
	}
	assertSourceAwareCandidates(t, warm, source.ID, 25)

	started := time.Now()
	candidates, err := catalog.RatedCandidates(ctx, 1400, 1600, nil, 25)
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(started)
	if elapsed >= 250*time.Millisecond {
		t.Fatalf("active candidate query took %s, want less than 250ms", elapsed)
	}
	assertSourceAwareCandidates(t, candidates, source.ID, 25)
	t.Logf("candidate_elapsed=%s active_occurrences=%d returned=%d", elapsed, syntheticPerformanceRows, len(candidates))
}

func TestRatedCandidateLimitAvoidsRankingWholeGeneration(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	ctx := context.Background()
	source := testSource("candidate-limit-regression", "synthetic", "/candidate-limit")
	active, _ := seedSyntheticSealedGeneration(
		t,
		catalog,
		store,
		source,
		"candidate-limit",
		syntheticPerformanceRows,
	)
	if err := active.Activate(ctx); err != nil {
		t.Fatal(err)
	}
	warmPuzzleReadPool(t, store.Reader)
	if _, err := catalog.RatedCandidates(ctx, 1400, 1600, nil, 1); err != nil {
		t.Fatal(err)
	}

	started := time.Now()
	candidates, err := catalog.RatedCandidates(ctx, 1400, 1600, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(started)
	if elapsed >= 50*time.Millisecond {
		t.Fatalf("one-candidate query took %s, want less than 50ms without ranking the full generation", elapsed)
	}
	assertSourceAwareCandidates(t, candidates, source.ID, 1)
	t.Logf("one_candidate_elapsed=%s active_occurrences=%d", elapsed, syntheticPerformanceRows)
}

func seedSyntheticSealedGeneration(
	t *testing.T,
	catalog *SQLiteCatalog,
	store *storage.PuzzleStore,
	source Source,
	fingerprintPrefix string,
	rowCount int,
) (GenerationImport, string) {
	t.Helper()
	ctx := context.Background()
	importing, err := catalog.BeginImport(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	var generationID string
	if err := store.Writer.QueryRowContext(
		ctx,
		`SELECT generation_id FROM source_generations WHERE source_path = ?`,
		source.Path,
	).Scan(&generationID); err != nil {
		t.Fatal(err)
	}

	tx, err := store.Writer.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	const chunkSize = 1_000
	for start := 1; start <= rowCount; start += chunkSize {
		end := min(start+chunkSize-1, rowCount)
		if _, err := tx.ExecContext(
			ctx,
			`WITH RECURSIVE sequence(value) AS (
			   SELECT ?
			   UNION ALL
			   SELECT value + 1 FROM sequence WHERE value < ?
			 )
			 INSERT INTO puzzle_cores(
			   fingerprint, displayed_fen, solver, solution_json, solution_plies
			 )
			 SELECT printf('%s-%07d', ?, value),
			        '7k/5Q2/6K1/8/8/8/8/8 w - - 0 1',
			        'white',
			        '[{"uci":"f7f8"}]',
			        1
			 FROM sequence`,
			start,
			end,
			fingerprintPrefix,
		); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`WITH RECURSIVE sequence(value) AS (
			   SELECT ?
			   UNION ALL
			   SELECT value + 1 FROM sequence WHERE value < ?
			 )
			 INSERT INTO puzzle_occurrences(
			   generation_id, fingerprint, external_id, source_fen, prelude_uci,
			   rating, popularity, play_count, source_url, attribution, metadata_json,
			   ordinal
			 )
			 SELECT ?,
			        printf('%s-%07d', ?, value),
			        printf('%s-%07d', ?, value),
			        'synthetic source FEN',
			        'e2e4',
			        1400 + (value % 200),
			        50,
			        1000,
			        'https://example.test/performance',
			        'performance test',
			        '{}',
			        value
			 FROM sequence`,
			start,
			end,
			generationID,
			fingerprintPrefix,
			fingerprintPrefix,
		); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`WITH RECURSIVE sequence(value) AS (
			   SELECT ?
			   UNION ALL
			   SELECT value + 1 FROM sequence WHERE value < ?
			 )
			 INSERT INTO occurrence_ratings(
			   generation_id, rating_key, fingerprint
			 )
			 SELECT ?,
			        1400 + (value % 200),
			        printf('%s-%07d', ?, value)
			 FROM sequence`,
			start,
			end,
			generationID,
			fingerprintPrefix,
		); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE source_generations
		 SET status = 'sealed', checksum = ?, sealed_at = ?
		 WHERE generation_id = ? AND status = 'building'`,
		fmt.Sprintf("synthetic-%s", fingerprintPrefix),
		time.Now().Unix(),
		generationID,
	); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	staged := importing.(*sqliteGenerationImport)
	if err := staged.stage.closeAndRemove(); err != nil {
		t.Fatal(err)
	}
	staged.stage = nil
	staged.state = generationImportSealed
	return importing, generationID
}

func assertSourceHead(t *testing.T, store *storage.PuzzleStore, sourceID, generationID string) {
	t.Helper()
	var got string
	if err := store.Reader.QueryRow(
		`SELECT generation_id FROM source_heads WHERE source_id = ?`,
		sourceID,
	).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != generationID {
		t.Fatalf("source head = %q, want %q", got, generationID)
	}
}

func assertGenerationOccurrenceCount(
	t *testing.T,
	store *storage.PuzzleStore,
	generationID string,
	want int,
) {
	t.Helper()
	var got int
	if err := store.Reader.QueryRow(
		`SELECT COUNT(*) FROM puzzle_occurrences WHERE generation_id = ?`,
		generationID,
	).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("generation %q occurrence rows = %d, want %d", generationID, got, want)
	}
}

func warmPuzzleReadPool(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	connections := make([]*sql.Conn, 0, 4)
	for range 4 {
		connection, err := db.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err := connection.PingContext(ctx); err != nil {
			connection.Close()
			t.Fatal(err)
		}
		connections = append(connections, connection)
	}
	for _, connection := range connections {
		if err := connection.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func assertSourceAwareCandidates(t *testing.T, candidates []TrainingPuzzle, sourceID string, want int) {
	t.Helper()
	if len(candidates) != want {
		t.Fatalf("candidate count = %d, want %d", len(candidates), want)
	}
	for index, candidate := range candidates {
		if candidate.Core.Fingerprint == "" || candidate.Occurrence.SourceID != sourceID ||
			candidate.Occurrence.SourceKind != "synthetic" {
			t.Fatalf("candidate %d is not source-aware: %+v", index, candidate)
		}
	}
}
