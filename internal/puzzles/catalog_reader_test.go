package puzzles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"testing"

	"chess-trainer/internal/domain"
	"chess-trainer/internal/storage"
)

func TestGetRequiresFingerprintAndSource(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	alpha := testSource("alpha", "csv", "/get-alpha")
	beta := testSource("beta", "lichess", "/get-beta")
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

func TestGetHydratesThemesFromOccurrenceSnapshot(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testSource("theme-snapshot", "test", "/theme-snapshot")
	generationID := seedActiveReaderGeneration(
		t,
		store,
		source,
		testTrainingPuzzle(source, "theme-snapshot-puzzle", 1500, "fork", "pin"),
	)

	if _, err := store.Writer.Exec(`UPDATE puzzle_occurrences
		SET themes_json = '["fork","pin"]'
		WHERE generation_id = ? AND fingerprint = ?`, generationID, "theme-snapshot-puzzle"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Writer.Exec(`DELETE FROM occurrence_themes
		WHERE generation_id = ? AND fingerprint = ?`, generationID, "theme-snapshot-puzzle"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Writer.Exec(`INSERT INTO occurrence_themes(
		generation_id, theme, fingerprint
	) VALUES (?, 'decoy', ?)`, generationID, "theme-snapshot-puzzle"); err != nil {
		t.Fatal(err)
	}

	got, err := catalog.Get(context.Background(), PuzzleKey{
		Fingerprint: "theme-snapshot-puzzle",
		SourceID:    source.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Occurrence.Themes, []string{"fork", "pin"}) {
		t.Fatalf("hydrated themes = %q, want occurrence snapshot", got.Occurrence.Themes)
	}
}

func TestResolvePrefersRequestedActiveSource(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	alpha := testSource("alpha", "csv", "/resolve-alpha")
	beta := testSource("beta", "lichess", "/resolve-beta")
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
	alpha := testSource("alpha", "csv", "/fallback-alpha")
	beta := testSource("beta", "lichess", "/fallback-beta")
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
	source := testSource("active", "test", "/active-head")
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
	abandonedSource := testSource("abandoned", "test", "/abandoned-head")
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
	alpha := testSource("alpha", "csv", "/rated-alpha")
	beta := testSource("beta", "csv", "/rated-beta")
	gamma := testSource("gamma", "csv", "/rated-gamma")
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

func TestRatedCandidatesRankAroundBandCenterBeforeLimit(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	alpha := testSource("alpha-ranked", "lichess", "/ranked-alpha")
	beta := testSource("beta-ranked", "lichess", "/ranked-beta")
	popularity10, popularity20, popularity50 := 10, 20, 50
	plays100, plays200 := 100, 200

	centerLowQuality := testTrainingPuzzle(alpha, "center-low-quality", 1500)
	centerLowQuality.Occurrence.Popularity = &popularity10
	centerLowQuality.Occurrence.PlayCount = &plays100
	centerHighPlay := testTrainingPuzzle(alpha, "center-high-play", 1500)
	centerHighPlay.Occurrence.Popularity = &popularity10
	centerHighPlay.Occurrence.PlayCount = &plays200
	centerHighPopularity := testTrainingPuzzle(alpha, "center-high-popularity", 1500)
	centerHighPopularity.Occurrence.Popularity = &popularity20
	centerHighPopularity.Occurrence.PlayCount = &plays100
	lowerNear := testTrainingPuzzle(alpha, "lower-near", 1499)
	lowerNear.Occurrence.Popularity = &popularity20
	upperNear := testTrainingPuzzle(beta, "upper-near", 1501)
	upperNear.Occurrence.Popularity = &popularity50
	bandBottom := testTrainingPuzzle(alpha, "band-bottom", 1400)
	for index, puzzle := range []*TrainingPuzzle{
		&centerLowQuality, &centerHighPlay, &centerHighPopularity, &lowerNear, &bandBottom,
	} {
		puzzle.Occurrence.Ordinal = int64(index + 1)
	}
	upperNear.Occurrence.Ordinal = 1
	sealAndActivate(t, beginGenerationImport(t, catalog, alpha),
		centerLowQuality, centerHighPlay, centerHighPopularity, lowerNear, bandBottom)
	sealAndActivate(t, beginGenerationImport(t, catalog, beta), upperNear)

	got, err := catalog.RatedCandidates(context.Background(), 1400, 1600, nil, 5)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"center-high-popularity",
		"center-high-play",
		"center-low-quality",
		"upper-near",
		"lower-near",
	}
	if fingerprints := candidateFingerprints(got); !reflect.DeepEqual(fingerprints, want) {
		t.Fatalf("ranked candidates = %q, want center distance then quality %q", fingerprints, want)
	}
}

func TestRatedCandidatesUseSourceThenFingerprintAsStableTieBreakers(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	alpha := testSource("alpha-tie", "lichess", "/tie-alpha")
	beta := testSource("beta-tie", "lichess", "/tie-beta")
	popularity, plays := 50, 1_000
	alphaPuzzle := testTrainingPuzzle(alpha, "z-alpha-fingerprint", 1500)
	alphaPuzzle.Occurrence.Popularity, alphaPuzzle.Occurrence.PlayCount = &popularity, &plays
	betaPuzzle := testTrainingPuzzle(beta, "a-beta-fingerprint", 1500)
	betaPuzzle.Occurrence.Popularity, betaPuzzle.Occurrence.PlayCount = &popularity, &plays
	sealAndActivate(t, beginGenerationImport(t, catalog, beta), betaPuzzle)
	sealAndActivate(t, beginGenerationImport(t, catalog, alpha), alphaPuzzle)

	got, err := catalog.RatedCandidates(context.Background(), 1400, 1600, nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprints := candidateFingerprints(got); !reflect.DeepEqual(
		fingerprints,
		[]string{"z-alpha-fingerprint", "a-beta-fingerprint"},
	) {
		t.Fatalf("stable candidate order = %q, want source then fingerprint", fingerprints)
	}
}

func TestRatedCandidatesAreDrivenByRatingMembership(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testSource("rated-membership", "test", "/rated-membership")
	included := testTrainingPuzzle(source, "rated-included", 1500)
	withoutMembership := testTrainingPuzzle(source, "rated-without-membership", 1500)
	withoutMembership.Occurrence.Ordinal = 2
	generationID := seedActiveReaderGeneration(
		t,
		store,
		source,
		included,
		withoutMembership,
	)

	if _, err := store.Writer.Exec(`DELETE FROM occurrence_ratings WHERE generation_id = ?`, generationID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Writer.Exec(`INSERT INTO occurrence_ratings(
		generation_id, rating_key, fingerprint
	) VALUES (?, 1500, 'rated-included')`, generationID); err != nil {
		t.Fatal(err)
	}

	got, err := catalog.RatedCandidates(context.Background(), 1400, 1600, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprints := candidateFingerprints(got); !reflect.DeepEqual(fingerprints, []string{"rated-included"}) {
		t.Fatalf("rated candidates = %q, want rating membership only", fingerprints)
	}
}

func TestActiveSourceSummaryRatingBoundsUseOrderedMembership(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testSource("summary-membership", "lichess", "/summary-membership")
	low := testTrainingPuzzle(source, "summary-low", 800)
	middle := testTrainingPuzzle(source, "summary-middle", 1500)
	high := testTrainingPuzzle(source, "summary-high", 2400)
	middle.Occurrence.Ordinal = 2
	high.Occurrence.Ordinal = 3
	generationID := seedActiveReaderGeneration(t, store, source, low, middle, high)
	if _, err := store.Writer.Exec(`DELETE FROM occurrence_ratings
		WHERE generation_id = ? AND fingerprint IN ('summary-low', 'summary-high')`, generationID); err != nil {
		t.Fatal(err)
	}

	summaries, err := catalog.ActiveSourceSummaries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 || summaries[0].MinimumRating == nil || summaries[0].MaximumRating == nil ||
		*summaries[0].MinimumRating != 1500 || *summaries[0].MaximumRating != 1500 {
		t.Fatalf("active summaries = %+v, want bounds from ordered rating membership", summaries)
	}
}

func TestLearnerRatingBoundsTrackActiveLichessGenerationAndFallback(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	ctx := context.Background()
	assertBounds := func(want RatingBounds) {
		t.Helper()
		got, err := catalog.LearnerRatingBounds(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("LearnerRatingBounds() = %+v, want %+v", got, want)
		}
	}
	assertBounds(DefaultLearnerRatingBounds())

	other := testSource("private", "pgn", "/private")
	sealAndActivate(t, beginGenerationImport(t, catalog, other), testTrainingPuzzle(other, "private-wide", 5000))
	assertBounds(DefaultLearnerRatingBounds())

	lichess := testSource("lichess", "lichess", "/lichess-old")
	unrated := testTrainingPuzzle(lichess, "lichess-unrated", 0)
	unrated.Occurrence.Rating = nil
	high := testTrainingPuzzle(lichess, "lichess-old-high", 1800)
	high.Occurrence.Ordinal = 2
	sealAndActivate(t, beginGenerationImport(t, catalog, lichess),
		testTrainingPuzzle(lichess, "lichess-old-low", 800), unrated, high)
	assertBounds(RatingBounds{Minimum: 800, Maximum: 1800})

	replacementSource := lichess
	replacementSource.Path = "/lichess-new"
	replacement := beginGenerationImport(t, catalog, replacementSource)
	newHigh := testTrainingPuzzle(lichess, "lichess-new-high", 2400)
	newHigh.Occurrence.Ordinal = 2
	if err := replacement.Add(ctx, testTrainingPuzzle(lichess, "lichess-new-low", 1000)); err != nil {
		t.Fatal(err)
	}
	if err := replacement.Add(ctx, newHigh); err != nil {
		t.Fatal(err)
	}
	if _, err := replacement.Seal(ctx, "new-checksum"); err != nil {
		t.Fatal(err)
	}
	assertBounds(RatingBounds{Minimum: 800, Maximum: 1800})
	if err := replacement.Activate(ctx); err != nil {
		t.Fatal(err)
	}
	assertBounds(RatingBounds{Minimum: 1000, Maximum: 2400})

	nullSource := lichess
	nullSource.Path = "/lichess-null"
	nullPuzzle := testTrainingPuzzle(lichess, "lichess-null", 0)
	nullPuzzle.Occurrence.Rating = nil
	sealAndActivate(t, beginGenerationImport(t, catalog, nullSource), nullPuzzle)
	assertBounds(DefaultLearnerRatingBounds())
}

func TestLearnerRatingBoundsDoNotReadOccurrencePayloadTables(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testSource("bounds-only", "lichess", "/bounds-only")
	high := testTrainingPuzzle(source, "bounds-high", 2200)
	high.Occurrence.Ordinal = 2
	seedActiveReaderGeneration(
		t,
		store,
		source,
		testTrainingPuzzle(source, "bounds-low", 900),
		high,
	)
	if _, err := store.Writer.Exec(`PRAGMA foreign_keys=OFF`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Writer.Exec(`DROP TABLE puzzle_occurrences`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Writer.Exec(`DROP TABLE puzzle_cores`); err != nil {
		t.Fatal(err)
	}

	got, err := catalog.LearnerRatingBounds(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if want := (RatingBounds{Minimum: 900, Maximum: 2200}); got != want {
		t.Fatalf("LearnerRatingBounds() = %+v, want %+v", got, want)
	}
}

func TestRatedCandidatesPreferOnlySourcesInsideRatingWindow(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	alpha := testSource("alpha", "csv", "/rated-window-alpha")
	beta := testSource("beta", "csv", "/rated-window-beta")
	// Alpha is lexically preferred for shared fingerprints, but its occurrence
	// is outside this request's rating window. Beta must remain eligible.
	sealAndActivate(t, beginGenerationImport(t, catalog, alpha), testTrainingPuzzle(alpha, "shared-window", 900))
	sealAndActivate(t, beginGenerationImport(t, catalog, beta), testTrainingPuzzle(beta, "shared-window", 1500))

	got, err := catalog.RatedCandidates(context.Background(), 1000, 2000, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Core.Fingerprint != "shared-window" ||
		got[0].Occurrence.SourceID != beta.ID || got[0].Occurrence.Rating == nil ||
		*got[0].Occurrence.Rating != 1500 {
		t.Fatalf("rated-window candidates = %+v, want beta's in-window occurrence", got)
	}
}

func TestFreePracticeThemeStrategySwitchesAboveOneThousandMemberships(t *testing.T) {
	for _, test := range []struct {
		name            string
		membershipCount int
		wantCandidates  int
	}{
		{name: "one thousand uses membership first", membershipCount: 1_000, wantCandidates: 1},
		{name: "above one thousand uses rating first", membershipCount: 1_001, wantCandidates: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			catalog, store := openTestGenerationalCatalog(t)
			source := testSource(
				fmt.Sprintf("theme-threshold-%d", test.membershipCount),
				"test",
				fmt.Sprintf("/theme-threshold-%d", test.membershipCount),
			)
			puzzles := make([]TrainingPuzzle, 0, test.membershipCount)
			for index := range test.membershipCount {
				puzzle := testTrainingPuzzle(
					source,
					fmt.Sprintf("threshold-%04d", index),
					1_200+index,
					"fork",
				)
				puzzle.Occurrence.Ordinal = int64(index + 1)
				puzzles = append(puzzles, puzzle)
			}
			generationID := seedActiveReaderGeneration(t, store, source, puzzles...)
			if _, err := store.Writer.Exec(`DELETE FROM occurrence_ratings WHERE generation_id = ?`, generationID); err != nil {
				t.Fatal(err)
			}

			got, err := catalog.FreePracticeCandidates(
				context.Background(),
				source.ID,
				nil,
				nil,
				[]string{"fork"},
				nil,
				1,
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != test.wantCandidates {
				t.Fatalf(
					"candidate count with %d memberships = %d, want %d",
					test.membershipCount,
					len(got),
					test.wantCandidates,
				)
			}
		})
	}
}

func TestFreePracticeCandidatesAreDrivenByRatingMembership(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testSource("practice-rating-membership", "test", "/practice-rating-membership")
	included := testTrainingPuzzle(source, "practice-included", 1400)
	withoutMembership := testTrainingPuzzle(source, "practice-without-membership", 1500)
	withoutMembership.Occurrence.Ordinal = 2
	generationID := seedActiveReaderGeneration(t, store, source, included, withoutMembership)
	if _, err := store.Writer.Exec(`DELETE FROM occurrence_ratings
		WHERE generation_id = ? AND fingerprint = 'practice-without-membership'`, generationID); err != nil {
		t.Fatal(err)
	}

	got, err := catalog.FreePracticeCandidates(
		context.Background(), source.ID, nil, nil, nil, nil, 10,
	)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprints := candidateFingerprints(got); !reflect.DeepEqual(fingerprints, []string{"practice-included"}) {
		t.Fatalf("practice candidates = %q, want rating membership only", fingerprints)
	}
}

func TestFreePracticeWithoutRatingBoundsIncludesUnratedOccurrences(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testSource("practice-unrated", "test", "/practice-unrated")
	unrated := testTrainingPuzzle(source, "practice-unrated", 0)
	unrated.Occurrence.Rating = nil
	rated := testTrainingPuzzle(source, "practice-rated", 1500)
	rated.Occurrence.Ordinal = 2
	seedActiveReaderGeneration(t, store, source, unrated, rated)

	got, err := catalog.FreePracticeCandidates(
		context.Background(), source.ID, nil, nil, nil, nil, 10,
	)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprints := candidateFingerprints(got); !reflect.DeepEqual(
		fingerprints,
		[]string{"practice-unrated", "practice-rated"},
	) {
		t.Fatalf("unbounded practice candidates = %q, want unrated first", fingerprints)
	}

	maximum := 2_000
	bounded, err := catalog.FreePracticeCandidates(
		context.Background(), source.ID, nil, &maximum, nil, nil, 10,
	)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprints := candidateFingerprints(bounded); !reflect.DeepEqual(
		fingerprints,
		[]string{"practice-rated"},
	) {
		t.Fatalf("maximum-bounded practice candidates = %q, want rated only", fingerprints)
	}
}

func TestFreePracticeOverlappingThemesPreserveAnySemantics(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testSource("theme-overlap", "test", "/theme-overlap")
	const puzzleCount = 501
	puzzles := make([]TrainingPuzzle, 0, puzzleCount)
	for index := range puzzleCount {
		puzzle := testTrainingPuzzle(
			source,
			fmt.Sprintf("overlap-%04d", index),
			1_200+index,
			"fork",
			"pin",
		)
		puzzle.Occurrence.Ordinal = int64(index + 1)
		puzzles = append(puzzles, puzzle)
	}
	generationID := seedActiveReaderGeneration(t, store, source, puzzles...)
	if _, err := store.Writer.Exec(`DELETE FROM occurrence_ratings
		WHERE generation_id = ? AND fingerprint <> 'overlap-0500'`, generationID); err != nil {
		t.Fatal(err)
	}

	got, err := catalog.FreePracticeCandidates(
		context.Background(),
		source.ID,
		nil,
		nil,
		[]string{"fork", "pin"},
		nil,
		10,
	)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprints := candidateFingerprints(got); !reflect.DeepEqual(fingerprints, []string{"overlap-0500"}) {
		t.Fatalf("overlapping-theme candidates = %q, want one eligible fingerprint", fingerprints)
	}
}

func TestReaderSeesOldHeadAcrossManyImportBatches(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	source := testSource("batched", "test", "/batched-active")
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
	source := testSource("snapshot", "test", "/snapshot-0")
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

func versionedSnapshotPuzzle(source Source, version int) TrainingPuzzle {
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

func seedActiveReaderGeneration(
	t *testing.T,
	store *storage.PuzzleStore,
	source Source,
	puzzles ...TrainingPuzzle,
) string {
	t.Helper()
	ctx := context.Background()
	generationID := source.ID + "-reader-generation"
	tx, err := store.Writer.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO sources(source_id, kind) VALUES (?, ?)`, source.ID, source.Kind); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO source_generations(
		generation_id, source_id, status, source_path, checksum, started_at, sealed_at
	) VALUES (?, ?, 'sealed', ?, 'reader-fixture', ?, ?)`,
		generationID, source.ID, source.Path, source.StartedAt.Unix(), source.StartedAt.Unix()); err != nil {
		t.Fatal(err)
	}
	for _, puzzle := range puzzles {
		solutionJSON, err := json.Marshal(normalizeNodes(puzzle.Core.Solution))
		if err != nil {
			t.Fatal(err)
		}
		metadata := puzzle.Occurrence.Metadata
		if metadata == nil {
			metadata = map[string]any{}
		}
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			t.Fatal(err)
		}
		themes := domain.NormalizeThemes(puzzle.Occurrence.Themes)
		themesJSON, err := json.Marshal(themes)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO puzzle_cores(
			fingerprint, displayed_fen, solver, solution_json, solution_plies
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(fingerprint) DO NOTHING`,
			puzzle.Core.Fingerprint,
			puzzle.Core.DisplayedFEN,
			puzzle.Core.Solver,
			string(solutionJSON),
			puzzle.Core.SolutionPlies,
		); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO puzzle_occurrences(
			generation_id, fingerprint, external_id, source_fen, prelude_uci,
			rating, popularity, play_count, source_url, attribution, metadata_json,
			themes_json, ordinal
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			generationID,
			puzzle.Core.Fingerprint,
			generationNullString(puzzle.Occurrence.ExternalID),
			generationNullString(puzzle.Occurrence.SourceFEN),
			generationNullString(puzzle.Occurrence.PreludeUCI),
			generationNullableInt(puzzle.Occurrence.Rating),
			generationNullableInt(puzzle.Occurrence.Popularity),
			generationNullableInt(puzzle.Occurrence.PlayCount),
			generationNullString(puzzle.Occurrence.URL),
			generationNullString(puzzle.Occurrence.Attribution),
			string(metadataJSON),
			string(themesJSON),
			puzzle.Occurrence.Ordinal,
		); err != nil {
			t.Fatal(err)
		}
		ratingKey := nullPuzzleRatingKey
		if puzzle.Occurrence.Rating != nil {
			ratingKey = int64(*puzzle.Occurrence.Rating)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO occurrence_ratings(
			generation_id, rating_key, fingerprint
		) VALUES (?, ?, ?)`, generationID, ratingKey, puzzle.Core.Fingerprint); err != nil {
			t.Fatal(err)
		}
		for _, theme := range themes {
			if _, err := tx.ExecContext(ctx, `INSERT INTO occurrence_themes(
				generation_id, theme, fingerprint
			) VALUES (?, ?, ?)`, generationID, theme, puzzle.Core.Fingerprint); err != nil {
				t.Fatal(err)
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO generation_themes(generation_id, theme)
				VALUES (?, ?) ON CONFLICT(generation_id, theme) DO NOTHING`, generationID, theme); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO source_heads(source_id, generation_id) VALUES (?, ?)`, source.ID, generationID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return generationID
}
