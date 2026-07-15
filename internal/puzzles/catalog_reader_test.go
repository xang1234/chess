package puzzles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"testing"

	"chess-trainer/internal/domain"
)

func TestGetRequiresFingerprintAndSource(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	alpha := testGenerationSource("alpha", "csv", "/get-alpha")
	beta := testGenerationSource("beta", "lichess", "/get-beta")
	alphaPuzzle := testTrainingPuzzle(alpha, "same", 1200, "fork")
	betaPuzzle := testTrainingPuzzle(beta, "same", 1800, "pin")
	sealAndActivate(t, beginGenerationImport(t, catalog, alpha), alphaPuzzle)
	sealAndActivate(t, beginGenerationImport(t, catalog, beta), betaPuzzle)

	got, err := catalog.Get(context.Background(), PuzzleKey{Fingerprint: "same", SourceID: "beta"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Occurrence.SourceID != "beta" || got.Occurrence.Rating == nil || *got.Occurrence.Rating != 1800 {
		t.Fatalf("Get(beta) = %+v", got.Occurrence)
	}
	for _, key := range []PuzzleKey{
		{Fingerprint: "same"},
		{SourceID: "beta"},
		{Fingerprint: "same", SourceID: "missing"},
	} {
		if _, err := catalog.Get(context.Background(), key); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("Get(%+v) err = %v, want sql.ErrNoRows", key, err)
		}
	}
}

func TestResolvePrefersRequestedActiveSource(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	alpha := testGenerationSource("alpha", "csv", "/resolve-alpha")
	beta := testGenerationSource("beta", "lichess", "/resolve-beta")
	sealAndActivate(t, beginGenerationImport(t, catalog, alpha), testTrainingPuzzle(alpha, "shared", 1200, "fork"))
	sealAndActivate(t, beginGenerationImport(t, catalog, beta), testTrainingPuzzle(beta, "shared", 1800, "pin"))

	got, err := catalog.Resolve(context.Background(), "shared", "beta")
	if err != nil {
		t.Fatal(err)
	}
	if got.Occurrence.SourceID != "beta" || got.Occurrence.Rating == nil || *got.Occurrence.Rating != 1800 {
		t.Fatalf("Resolve(preferred beta) = %+v", got.Occurrence)
	}
}

func TestResolveFallsBackLexicographically(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	alpha := testGenerationSource("alpha", "csv", "/fallback-alpha")
	beta := testGenerationSource("beta", "lichess", "/fallback-beta")
	sealAndActivate(t, beginGenerationImport(t, catalog, beta), testTrainingPuzzle(beta, "shared", 1800, "pin"))
	sealAndActivate(t, beginGenerationImport(t, catalog, alpha), testTrainingPuzzle(alpha, "shared", 1200, "fork"))

	for _, preferred := range []string{"", "missing"} {
		got, err := catalog.Resolve(context.Background(), "shared", preferred)
		if err != nil {
			t.Fatal(err)
		}
		if got.Occurrence.SourceID != "alpha" {
			t.Fatalf("Resolve(preferred %q) source = %q, want alpha", preferred, got.Occurrence.SourceID)
		}
	}
}

func TestCandidatesAndMetadataUseOnlyActiveHeads(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	source := testGenerationSource("active", "test", "/active-head")
	active := testTrainingPuzzle(source, "active-fingerprint", 1500, "active-theme")
	active.Core.Solution = []domain.MoveNode{{UCI: "f7f8", Children: []domain.MoveNode{{UCI: "h8h7"}}}}
	active.Core.SolutionPlies = 2
	sealAndActivate(t, beginGenerationImport(t, catalog, source), active)

	supersededSource := source
	supersededSource.Path = "/sealed-unheaded"
	superseded := beginGenerationImport(t, catalog, supersededSource)
	if err := superseded.Add(context.Background(), testTrainingPuzzle(source, "sealed-only", 1550, "sealed-theme")); err != nil {
		t.Fatal(err)
	}
	if _, err := superseded.Seal(context.Background(), "sealed-checksum"); err != nil {
		t.Fatal(err)
	}
	abandonedSource := testGenerationSource("abandoned", "test", "/abandoned-head")
	abandoned := beginGenerationImport(t, catalog, abandonedSource)
	if err := abandoned.Add(context.Background(), testTrainingPuzzle(abandonedSource, "abandoned-only", 1525, "abandoned-theme")); err != nil {
		t.Fatal(err)
	}
	if err := abandoned.Abandon(context.Background()); err != nil {
		t.Fatal(err)
	}

	rated, err := catalog.RatedCandidates(context.Background(), 1400, 1600, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rated) != 1 || rated[0].Core.Fingerprint != "active-fingerprint" ||
		!reflect.DeepEqual(rated[0].Occurrence.Themes, []string{"active-theme"}) {
		t.Fatalf("rated candidates = %+v, want only active head", rated)
	}
	minimum, maximum, maximumPlies := 1400, 1600, 2
	practice, err := catalog.FreePracticeCandidates(
		context.Background(),
		source.ID,
		&minimum,
		&maximum,
		[]string{"active-theme"},
		&maximumPlies,
		10,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(practice) != 1 || practice[0].Core.Fingerprint != "active-fingerprint" {
		t.Fatalf("practice candidates = %+v, want only active head", practice)
	}
	summaries, err := catalog.ActiveSourceSummaries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 || summaries[0].SourceID != source.ID || summaries[0].Kind != source.Kind ||
		summaries[0].MinimumRating == nil || *summaries[0].MinimumRating != 1500 ||
		summaries[0].MaximumRating == nil || *summaries[0].MaximumRating != 1500 ||
		summaries[0].MaximumSolutionPlies != 2 {
		t.Fatalf("active summaries = %+v", summaries)
	}
	themes, err := catalog.ActiveThemes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(themes, []string{"active-theme"}) {
		t.Fatalf("active themes = %q, want only active-theme", themes)
	}

	if empty, err := catalog.RatedCandidates(context.Background(), 0, 3000, nil, 0); err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("zero-limit rated = %#v, err=%v; want non-nil empty", empty, err)
	}
	if empty, err := catalog.FreePracticeCandidates(context.Background(), source.ID, nil, nil, nil, nil, -1); err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("negative-limit practice = %#v, err=%v; want non-nil empty", empty, err)
	}
}

func TestRatedCandidatesReturnOneDeterministicOccurrencePerFingerprint(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	alpha := testGenerationSource("alpha", "csv", "/rated-alpha")
	beta := testGenerationSource("beta", "csv", "/rated-beta")
	gamma := testGenerationSource("gamma", "csv", "/rated-gamma")
	alphaShared := testTrainingPuzzle(alpha, "shared", 1500, "alpha")
	alphaUnique := testTrainingPuzzle(alpha, "unique", 1600, "unique")
	betaShared := testTrainingPuzzle(beta, "shared", 1200, "beta")
	gammaShared := testTrainingPuzzle(gamma, "shared", 1300, "gamma")
	sealAndActivate(t, beginGenerationImport(t, catalog, beta), betaShared)
	sealAndActivate(t, beginGenerationImport(t, catalog, gamma), gammaShared)
	sealAndActivate(t, beginGenerationImport(t, catalog, alpha), alphaShared, alphaUnique)

	got, err := catalog.RatedCandidates(context.Background(), 1000, 2000, nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("rated candidate count = %d, want 2: %+v", len(got), got)
	}
	seen := make(map[string]TrainingPuzzle, len(got))
	for _, puzzle := range got {
		seen[puzzle.Core.Fingerprint] = puzzle
	}
	if len(seen) != 2 || seen["shared"].Occurrence.SourceID != "alpha" ||
		seen["shared"].Occurrence.Rating == nil || *seen["shared"].Occurrence.Rating != 1500 {
		t.Fatalf("rated candidates did not deduplicate before LIMIT with lexical source: %+v", got)
	}
	excluded, err := catalog.RatedCandidates(context.Background(), 1000, 2000, []string{"shared"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(excluded) != 1 || excluded[0].Core.Fingerprint != "unique" {
		t.Fatalf("excluded rated candidates = %+v, want unique", excluded)
	}
}

func TestReaderSeesOldHeadAcrossManyImportBatches(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	source := testGenerationSource("batched", "test", "/batched-active")
	sealAndActivate(t, beginGenerationImport(t, catalog, source), testTrainingPuzzle(source, "old-head", 1200, "old"))
	replacementSource := source
	replacementSource.Path = "/batched-building"
	replacement := beginGenerationImport(t, catalog, replacementSource)

	for index := range 2_100 {
		puzzle := testTrainingPuzzle(source, fmt.Sprintf("new-%04d", index), 1300, "new")
		puzzle.Occurrence.Ordinal = int64(index + 1)
		if err := replacement.Add(context.Background(), puzzle); err != nil {
			t.Fatal(err)
		}
		if index%250 == 0 {
			got, err := catalog.Get(context.Background(), PuzzleKey{Fingerprint: "old-head", SourceID: source.ID})
			if err != nil || got.Occurrence.ExternalID != "old-head" {
				t.Fatalf("old head during batch %d = %+v, err=%v", index, got, err)
			}
		}
	}
	if _, err := replacement.Seal(context.Background(), "replacement-checksum"); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Get(context.Background(), PuzzleKey{Fingerprint: "new-0000", SourceID: source.ID}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("sealed unheaded replacement is visible: %v", err)
	}
	if _, err := catalog.Get(context.Background(), PuzzleKey{Fingerprint: "old-head", SourceID: source.ID}); err != nil {
		t.Fatalf("old head disappeared before activation: %v", err)
	}
}

func TestReaderNeverMixesOccurrenceAcrossActivationAndCleanup(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testGenerationSource("snapshot", "test", "/snapshot-0")
	sealAndActivate(t, beginGenerationImport(t, catalog, source), versionedSnapshotPuzzle(source, 0))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errorsSeen := make(chan error, 1)
	var readers sync.WaitGroup
	for range 4 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				puzzle, err := catalog.Get(ctx, PuzzleKey{Fingerprint: "snapshot-core", SourceID: source.ID})
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					reportSnapshotReadError(errorsSeen, fmt.Errorf("read snapshot: %w", err))
					return
				}
				if err := validateSnapshotPuzzle(puzzle); err != nil {
					reportSnapshotReadError(errorsSeen, err)
					return
				}
			}
		}()
	}

	for version := 1; version <= 20; version++ {
		replacement := source
		replacement.Path = fmt.Sprintf("/snapshot-%d", version)
		importing := beginGenerationImport(t, catalog, replacement)
		if err := importing.Add(context.Background(), versionedSnapshotPuzzle(source, version)); err != nil {
			t.Fatal(err)
		}
		if _, err := importing.Seal(context.Background(), fmt.Sprintf("checksum-%d", version)); err != nil {
			t.Fatal(err)
		}
		if err := importing.Activate(context.Background()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Writer.Exec(`DELETE FROM source_generations
			WHERE status IN ('sealed', 'abandoned')
			  AND NOT EXISTS (
			    SELECT 1 FROM source_heads
			    WHERE source_heads.generation_id = source_generations.generation_id
			  )`); err != nil {
			t.Fatal(err)
		}
		select {
		case err := <-errorsSeen:
			t.Fatal(err)
		default:
		}
	}
	cancel()
	readers.Wait()
	select {
	case err := <-errorsSeen:
		t.Fatal(err)
	default:
	}
}

func versionedSnapshotPuzzle(source GenerationSource, version int) TrainingPuzzle {
	marker := strconv.Itoa(version)
	puzzle := testTrainingPuzzle(source, "snapshot-core", 1_000+version, "theme-"+marker)
	puzzle.Occurrence.ExternalID = "external-" + marker
	puzzle.Occurrence.SourceFEN = "source-fen-" + marker
	puzzle.Occurrence.PreludeUCI = "prelude-" + marker
	puzzle.Occurrence.URL = "url-" + marker
	puzzle.Occurrence.Attribution = "attribution-" + marker
	puzzle.Occurrence.Metadata = map[string]any{"version": marker}
	return puzzle
}

func validateSnapshotPuzzle(puzzle TrainingPuzzle) error {
	marker, ok := puzzle.Occurrence.Metadata["version"].(string)
	if !ok {
		return fmt.Errorf("snapshot metadata marker = %#v", puzzle.Occurrence.Metadata)
	}
	wantRating, err := strconv.Atoi(marker)
	if err != nil {
		return fmt.Errorf("parse snapshot marker %q: %w", marker, err)
	}
	wantRating += 1_000
	if puzzle.Occurrence.ExternalID != "external-"+marker ||
		puzzle.Occurrence.SourceFEN != "source-fen-"+marker ||
		puzzle.Occurrence.PreludeUCI != "prelude-"+marker ||
		puzzle.Occurrence.URL != "url-"+marker ||
		puzzle.Occurrence.Attribution != "attribution-"+marker ||
		puzzle.Occurrence.Rating == nil || *puzzle.Occurrence.Rating != wantRating ||
		!reflect.DeepEqual(puzzle.Occurrence.Themes, []string{"theme-" + marker}) {
		return fmt.Errorf("mixed snapshot for marker %q: %+v", marker, puzzle.Occurrence)
	}
	return nil
}

func reportSnapshotReadError(target chan<- error, err error) {
	select {
	case target <- err:
	default:
	}
}
