//go:build performance

package puzzles

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

const generationWinnerTracerRows = 500_000

func TestGenerationWinnerLargeSyntheticTracer(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	ctx := context.Background()
	importing := beginGenerationImport(
		t,
		catalog,
		testSource("winner-large-tracer", "synthetic", "/winner-large-tracer"),
	)
	staged := importing.(*sqliteGenerationImport)
	defer func() {
		if err := importing.Abandon(context.Background()); err != nil {
			t.Errorf("abandon winner tracer: %v", err)
		}
	}()

	appendStarted := time.Now()
	for start := 0; start < generationWinnerTracerRows; start += generationImportBatchSize {
		batch := make([]generationRow, 0, generationImportBatchSize)
		for offset := range generationImportBatchSize {
			index := start + offset
			fingerprintIndex := index
			if index%10_000 == 9_999 {
				fingerprintIndex--
			}
			rating := 1_200 + index%800
			batch = append(batch, generationRow{
				fingerprint:   fmt.Sprintf("winner-tracer-%07d", fingerprintIndex),
				displayedFEN:  "7k/5Q2/6K1/8/8/8/8/8 w - - 0 1",
				solver:        "white",
				solutionJSON:  `[{"uci":"f7f8"}]`,
				solutionPlies: 1,
				externalID:    fmt.Sprintf("external-%07d", index),
				sourceFEN:     "synthetic source FEN",
				preludeUCI:    "e2e4",
				rating:        &rating,
				popularity:    &rating,
				playCount:     &rating,
				sourceURL:     "https://example.test/winner-tracer",
				attribution:   "synthetic tracer",
				metadataJSON:  `{"synthetic":true}`,
				themes:        []string{"common", fmt.Sprintf("bucket-%02d", index%32)},
				ordinal:       int64(index + 1),
			})
		}
		if err := staged.stage.append(ctx, batch); err != nil {
			t.Fatal(err)
		}
	}
	appendElapsed := time.Since(appendStarted)
	appendPath := staged.stage.path

	compactStarted := time.Now()
	winner, stagedRows, winners, err := staged.compactToWinner(ctx)
	if err != nil {
		t.Fatal(err)
	}
	compactElapsed := time.Since(compactStarted)
	t.Cleanup(func() {
		if err := winner.closeAndRemove(); err != nil {
			t.Errorf("remove winner tracer artifact: %v", err)
		}
	})
	const duplicates = generationWinnerTracerRows / 10_000
	if stagedRows != generationWinnerTracerRows || winners != generationWinnerTracerRows-duplicates {
		t.Fatalf(
			"winner tracer counts staged=%d winners=%d, want %d/%d",
			stagedRows,
			winners,
			generationWinnerTracerRows,
			generationWinnerTracerRows-duplicates,
		)
	}
	if _, err := os.Stat(appendPath); !os.IsNotExist(err) {
		t.Fatalf("large append stage after compaction stat = %v, want not-exist", err)
	}

	staged.report.Accepted = winners
	staged.report.Duplicates = stagedRows - winners
	materializeStarted := time.Now()
	if err := staged.materializeWinner(ctx, winner); err != nil {
		t.Fatal(err)
	}
	materializeElapsed := time.Since(materializeStarted)
	var cores, occurrences, ratings, themes int64
	if err := store.Reader.QueryRow(`SELECT
		(SELECT COUNT(*) FROM puzzle_cores),
		(SELECT COUNT(*) FROM puzzle_occurrences WHERE generation_id = ?),
		(SELECT COUNT(*) FROM occurrence_ratings WHERE generation_id = ?),
		(SELECT COUNT(*) FROM occurrence_themes WHERE generation_id = ?)`,
		staged.generationID,
		staged.generationID,
		staged.generationID,
	).Scan(&cores, &occurrences, &ratings, &themes); err != nil {
		t.Fatal(err)
	}
	if cores != winners || occurrences != winners || ratings != winners || themes != winners*2 {
		t.Fatalf(
			"materialized rows cores=%d occurrences=%d ratings=%d themes=%d, want %d/%d/%d/%d",
			cores,
			occurrences,
			ratings,
			themes,
			winners,
			winners,
			winners,
			winners*2,
		)
	}
	mainPath, err := puzzleDatabasePath(ctx, store.Writer)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf(
		"winner_tracer_rows=%d winners=%d duplicates=%d append=%s compact=%s materialize=%s total=%s catalog_bytes=%d",
		stagedRows,
		winners,
		stagedRows-winners,
		appendElapsed,
		compactElapsed,
		materializeElapsed,
		appendElapsed+compactElapsed+materializeElapsed,
		catalogBytes(t, mainPath),
	)
}
