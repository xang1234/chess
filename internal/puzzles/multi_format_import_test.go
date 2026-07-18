package puzzles

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/domain"
	"chess-trainer/internal/storage"
)

const multiFormatSharedPGN = `[Event "Shared core"]
[SourceId "club-pgn"]
[PuzzleId "pgn-shared"]
[SetUp "1"]
[FEN "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"]
[White "solver"]
[Black "?"]

1. e4 Kf7 *
`

const multiFormatSharedJSON = `{
  "schema": "chess-trainer-puzzles/v1",
  "source": {"id": "club-json", "name": "Club JSON"},
  "puzzles": [{
    "id": "json-shared",
    "displayedFen": "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1",
    "solver": "white",
    "solution": [{"uci": "e2e4", "children": [{"uci": "e8f7", "children": []}]}],
    "rating": 1450,
    "themes": ["fork"]
  }]
}`

const multiFormatReplacementJSON = `{
  "schema": "chess-trainer-puzzles/v1",
  "source": {"id": "club-json", "name": "Club JSON replacement"},
  "puzzles": [{
    "id": "json-replacement",
    "displayedFen": "7k/P7/8/8/8/8/8/K7 w - - 0 1",
    "solver": "white",
    "solution": [{"uci": "a7a8q", "children": []}],
    "rating": 1650,
    "themes": ["promotion"]
  }]
}`

const multiFormatLucas = `4k3/8/8/8/8/8/4P3/4K3 w - - 0 1|Difficulty **|1. e4 Kf7 (1... Kd7) 2. Kf2 *`

const multiFormatLinear = `4k3/4p3/8/8/8/8/8/4K3 b - - 0 1 e7e5 e1f2 1750`

func TestMultiFormatImportPersistsEveryFormatAndGuidesOnlyExplicitRatings(t *testing.T) {
	catalog, store, importer, fixtureRoot := newMultiFormatTestCatalogue(t)
	ctx := context.Background()

	pgnPath := writeMultiFormatFixture(t, fixtureRoot, "club.pgn", multiFormatSharedPGN)
	jsonPath := writeMultiFormatFixture(t, fixtureRoot, "club.json", multiFormatSharedJSON)
	lucasPath := writeMultiFormatFixture(t, fixtureRoot, "Pin.fns", multiFormatLucas)
	linearPath := writeMultiFormatFixture(t, fixtureRoot, "linear.txt", multiFormatLinear)

	pgnInspection := importMultiFormatFixture(t, importer, pgnPath)
	jsonInspection := importMultiFormatFixture(t, importer, jsonPath)
	lucasInspection := importMultiFormatFixture(t, importer, lucasPath)
	linearInspection := importMultiFormatFixture(t, importer, linearPath)

	pgn := requireMultiFormatPuzzle(t, catalog, pgnInspection.SourceID, "pgn-shared")
	jsonPuzzle := requireMultiFormatPuzzle(t, catalog, jsonInspection.SourceID, "json-shared")
	lucas := requireMultiFormatPuzzle(t, catalog, lucasInspection.SourceID, "1")
	linear := requireMultiFormatPuzzle(t, catalog, linearInspection.SourceID, "1")

	sharedSolution := []domain.MoveNode{{
		UCI:      "e2e4",
		Children: []domain.MoveNode{{UCI: "e8f7"}},
	}}
	if pgn.Core.Fingerprint != jsonPuzzle.Core.Fingerprint ||
		!reflect.DeepEqual(pgn.Core.Solution, sharedSolution) ||
		!reflect.DeepEqual(jsonPuzzle.Core.Solution, sharedSolution) {
		t.Fatalf("shared PGN/JSON cores = %+v / %+v, want one normalized solution", pgn.Core, jsonPuzzle.Core)
	}
	if pgn.Occurrence.SourceKind != string(FormatTacticalPGN) ||
		pgn.Occurrence.Rating != nil || pgn.Occurrence.SourceID != "club-pgn" {
		t.Fatalf("PGN occurrence = %+v", pgn.Occurrence)
	}
	if jsonPuzzle.Occurrence.SourceKind != string(FormatCanonicalJSON) ||
		jsonPuzzle.Occurrence.Rating == nil || *jsonPuzzle.Occurrence.Rating != 1450 ||
		jsonPuzzle.Occurrence.SourceID != "club-json" {
		t.Fatalf("JSON occurrence = %+v", jsonPuzzle.Occurrence)
	}

	wantLucasSolution := []domain.MoveNode{{
		UCI: "e2e4",
		Children: []domain.MoveNode{
			{UCI: "e8f7", Children: []domain.MoveNode{{UCI: "e1f2"}}},
			{UCI: "e8d7"},
		},
	}}
	if lucas.Occurrence.SourceKind != string(FormatLucasFNS) ||
		lucas.Occurrence.SourceID != lucasInspection.Path || lucas.Occurrence.Rating != nil ||
		lucas.Occurrence.Metadata["sourceDifficulty"] != "**" ||
		!reflect.DeepEqual(lucas.Core.Solution, wantLucasSolution) {
		t.Fatalf("Lucas puzzle = %+v", lucas)
	}
	wantLinearSolution := []domain.MoveNode{{
		UCI:      "e7e5",
		Children: []domain.MoveNode{{UCI: "e1f2"}},
	}}
	if linear.Occurrence.SourceKind != string(FormatLinearFENUCI) ||
		linear.Occurrence.SourceID != linearInspection.Path || linear.Occurrence.Rating != nil ||
		linear.Occurrence.Metadata["sourceDifficulty"] != float64(1750) ||
		!reflect.DeepEqual(linear.Core.Solution, wantLinearSolution) {
		t.Fatalf("linear puzzle = %+v", linear)
	}

	summaries, err := catalog.ActiveSourceSummaries(ctx)
	if err != nil {
		t.Fatal(err)
	}
	bySource := make(map[string]SourceSummary, len(summaries))
	for _, summary := range summaries {
		bySource[summary.SourceID] = summary
	}
	for sourceID, wantKind := range map[string]string{
		pgnInspection.SourceID:    string(FormatTacticalPGN),
		jsonInspection.SourceID:   string(FormatCanonicalJSON),
		lucasInspection.SourceID:  string(FormatLucasFNS),
		linearInspection.SourceID: string(FormatLinearFENUCI),
	} {
		if summary, present := bySource[sourceID]; !present || summary.Kind != wantKind {
			t.Fatalf("active summary %q = %+v, present=%v, want kind %q", sourceID, summary, present, wantKind)
		}
	}
	if summary := bySource[jsonInspection.SourceID]; summary.MinimumRating == nil || *summary.MinimumRating != 1450 ||
		summary.MaximumRating == nil || *summary.MaximumRating != 1450 {
		t.Fatalf("rated JSON summary = %+v", summary)
	}
	for _, sourceID := range []string{
		pgnInspection.SourceID, lucasInspection.SourceID, linearInspection.SourceID,
	} {
		if summary := bySource[sourceID]; summary.MinimumRating != nil || summary.MaximumRating != nil {
			t.Fatalf("unrated source %q summary = %+v", sourceID, summary)
		}
	}

	var coreRows, activeOccurrences int
	if err := store.Reader.QueryRow(
		`SELECT COUNT(*) FROM puzzle_cores WHERE fingerprint = ?`,
		pgn.Core.Fingerprint,
	).Scan(&coreRows); err != nil {
		t.Fatal(err)
	}
	if err := store.Reader.QueryRow(
		`SELECT COUNT(*)
		 FROM puzzle_occurrences AS occurrence
		 JOIN source_heads AS head
		   ON head.generation_id = occurrence.generation_id
		 WHERE occurrence.fingerprint = ?`,
		pgn.Core.Fingerprint,
	).Scan(&activeOccurrences); err != nil {
		t.Fatal(err)
	}
	if coreRows != 1 || activeOccurrences != 2 {
		t.Fatalf("shared fingerprint core/active occurrence rows = %d/%d, want 1/2", coreRows, activeOccurrences)
	}

	candidates, err := catalog.RatedCandidates(ctx, 100, 4000, nil, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].Occurrence.SourceID != jsonInspection.SourceID ||
		candidates[0].Occurrence.Rating == nil || *candidates[0].Occurrence.Rating != 1450 {
		t.Fatalf("rated candidates = %+v, want only canonical JSON", candidates)
	}
	bounds, err := catalog.LearnerRatingBounds(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if bounds != (RatingBounds{Minimum: 1450, Maximum: 1450}) {
		t.Fatalf("learner bounds = %+v, want 1450..1450", bounds)
	}
}

func TestMultiFormatImportReplacesOnlyTheResolvedJSONSource(t *testing.T) {
	catalog, store, importer, fixtureRoot := newMultiFormatTestCatalogue(t)
	ctx := context.Background()

	pgnInspection := importMultiFormatFixture(
		t, importer, writeMultiFormatFixture(t, fixtureRoot, "shared.pgn", multiFormatSharedPGN),
	)
	jsonInspection := importMultiFormatFixture(
		t, importer, writeMultiFormatFixture(t, fixtureRoot, "shared.json", multiFormatSharedJSON),
	)
	oldPGN := requireMultiFormatPuzzle(t, catalog, pgnInspection.SourceID, "pgn-shared")
	oldJSON := requireMultiFormatPuzzle(t, catalog, jsonInspection.SourceID, "json-shared")
	oldPGNHead := multiFormatHeadID(t, store, pgnInspection.SourceID)
	oldJSONHead := multiFormatHeadID(t, store, jsonInspection.SourceID)

	replacementPath := writeMultiFormatFixture(
		t, fixtureRoot, "replacement.json", multiFormatReplacementJSON,
	)
	replacementInspection := importMultiFormatFixture(t, importer, replacementPath)
	if replacementInspection.SourceID != jsonInspection.SourceID || !replacementInspection.ReplacesExisting {
		t.Fatalf("replacement inspection = %+v", replacementInspection)
	}
	newJSONHead := multiFormatHeadID(t, store, jsonInspection.SourceID)
	if got := multiFormatHeadID(t, store, pgnInspection.SourceID); got != oldPGNHead {
		t.Fatalf("PGN head changed from %q to %q during JSON replacement", oldPGNHead, got)
	}
	if newJSONHead == oldJSONHead {
		t.Fatalf("JSON head remained %q after replacement", oldJSONHead)
	}
	if _, err := catalog.Get(ctx, oldPGN.Key()); err != nil {
		t.Fatalf("PGN shared occurrence disappeared: %v", err)
	}
	if _, err := catalog.Get(ctx, oldJSON.Key()); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("old JSON occurrence error = %v, want sql.ErrNoRows", err)
	}
	replacement := requireMultiFormatPuzzle(t, catalog, jsonInspection.SourceID, "json-replacement")
	if replacement.Occurrence.Rating == nil || *replacement.Occurrence.Rating != 1650 ||
		!reflect.DeepEqual(replacement.Core.Solution, []domain.MoveNode{{UCI: "a7a8q"}}) {
		t.Fatalf("replacement JSON puzzle = %+v", replacement)
	}
}

func TestMultiFormatImportFailuresPreserveThePriorPGNHead(t *testing.T) {
	catalog, store, importer, fixtureRoot := newMultiFormatTestCatalogue(t)
	ctx := context.Background()

	stableInspection := importMultiFormatFixture(
		t, importer, writeMultiFormatFixture(t, fixtureRoot, "stable.pgn", multiFormatSharedPGN),
	)
	stable := requireMultiFormatPuzzle(t, catalog, stableInspection.SourceID, "pgn-shared")
	stableHead := multiFormatHeadID(t, store, stableInspection.SourceID)

	zeroValid := `[SourceId "club-pgn"]
[SetUp "0"]
[FEN "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"]
[White "solver"]
[Black "?"]

1. e4 *
`
	zeroPath := writeMultiFormatFixture(t, fixtureRoot, "zero-valid.pgn", zeroValid)
	zeroInspection, err := importer.Inspect(ctx, zeroPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := importer.ImportFormat(
		ctx, zeroInspection.Format, zeroInspection.SourceID, zeroInspection.Path, nil,
	); !errors.Is(err, ErrNoValidPuzzles) {
		t.Fatalf("zero-valid import error = %v, want ErrNoValidPuzzles", err)
	}
	assertMultiFormatHeadUnchanged(t, catalog, store, stable, stableHead)

	cancelPath := writeMultiFormatFixture(t, fixtureRoot, "cancelled.pgn", multiFormatSharedPGN)
	cancelInspection, err := importer.Inspect(ctx, cancelPath)
	if err != nil {
		t.Fatal(err)
	}
	cancelContext, cancel := context.WithCancel(ctx)
	_, err = importer.ImportFormat(
		cancelContext,
		cancelInspection.Format,
		cancelInspection.SourceID,
		cancelInspection.Path,
		func(progress Progress) {
			if progress.Phase == ImportParsing {
				cancel()
			}
		},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled import error = %v, want context.Canceled", err)
	}
	assertMultiFormatHeadUnchanged(t, catalog, store, stable, stableHead)

	lateConflict := multiFormatSharedPGN + `
[Event "Late conflict"]
[SourceId "other-club"]
[PuzzleId "conflict"]
[SetUp "1"]
[FEN "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"]
[White "solver"]
[Black "?"]

1. e4 Kf7 *
`
	conflictPath := writeMultiFormatFixture(t, fixtureRoot, "conflict.pgn", lateConflict)
	conflictInspection, err := importer.Inspect(ctx, conflictPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := importer.ImportFormat(
		ctx,
		conflictInspection.Format,
		conflictInspection.SourceID,
		conflictInspection.Path,
		nil,
	); err == nil || !strings.Contains(err.Error(), "conflicts with inspected source ID") {
		t.Fatalf("late-conflicting import error = %v", err)
	}
	assertMultiFormatHeadUnchanged(t, catalog, store, stable, stableHead)
}

func newMultiFormatTestCatalogue(
	t *testing.T,
) (*SQLiteCatalog, *storage.PuzzleStore, CollectionImporter, string) {
	t.Helper()
	catalog, store := openTestGenerationalCatalog(t)
	fixtureRoot := t.TempDir()
	rules := chessrules.Rules{}
	importer := CollectionImporter{
		Catalog: catalog,
		Reader:  catalog,
		Adapters: []PuzzleAdapter{
			NewTacticalPGNAdapter(rules),
			NewCanonicalJSONAdapter(rules),
			NewLucasFNSAdapter(rules),
			NewLinearFENAdapter(rules),
		},
		CatalogDirectory: fixtureRoot,
		AvailableBytes: func(string) (uint64, error) {
			return math.MaxUint64, nil
		},
	}
	return catalog, store, importer, fixtureRoot
}

func writeMultiFormatFixture(t *testing.T, root, name, contents string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func importMultiFormatFixture(
	t *testing.T,
	importer CollectionImporter,
	path string,
) ImportInspection {
	t.Helper()
	inspection, err := importer.Inspect(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	report, err := importer.ImportFormat(
		context.Background(), inspection.Format, inspection.SourceID, inspection.Path, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.Accepted != 1 || report.Duplicates != 0 || report.Rejected != 0 {
		t.Fatalf("import %s report = %+v, want one accepted puzzle", path, report)
	}
	return inspection
}

func requireMultiFormatPuzzle(
	t *testing.T,
	catalog *SQLiteCatalog,
	sourceID string,
	externalID string,
) TrainingPuzzle {
	t.Helper()
	var fingerprint string
	err := catalog.readDB.QueryRow(
		`SELECT occurrence.fingerprint
		 FROM source_heads AS head
		 JOIN puzzle_occurrences AS occurrence
		   ON occurrence.generation_id = head.generation_id
		 WHERE head.source_id = ? AND occurrence.external_id = ?`,
		sourceID,
		externalID,
	).Scan(&fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	puzzle, err := catalog.Get(context.Background(), PuzzleKey{
		Fingerprint: fingerprint,
		SourceID:    sourceID,
	})
	if err != nil {
		t.Fatal(err)
	}
	return puzzle
}

func multiFormatHeadID(t *testing.T, store *storage.PuzzleStore, sourceID string) string {
	t.Helper()
	var generationID string
	if err := store.Reader.QueryRow(
		`SELECT generation_id FROM source_heads WHERE source_id = ?`, sourceID,
	).Scan(&generationID); err != nil {
		t.Fatal(err)
	}
	return generationID
}

func assertMultiFormatHeadUnchanged(
	t *testing.T,
	catalog *SQLiteCatalog,
	store *storage.PuzzleStore,
	want TrainingPuzzle,
	wantHead string,
) {
	t.Helper()
	if gotHead := multiFormatHeadID(t, store, want.Occurrence.SourceID); gotHead != wantHead {
		t.Fatalf("source head changed from %q to %q", wantHead, gotHead)
	}
	got, err := catalog.Get(context.Background(), want.Key())
	if err != nil {
		t.Fatalf("prior active puzzle is not queryable: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("prior active puzzle changed:\n got: %+v\nwant: %+v", got, want)
	}
}
