package puzzles

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"testing"
)

func TestFreePracticeCandidatesOrderByRatingThenFingerprint(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	source := testSource("practice-order", "test", "/practice-order")
	low := testTrainingPuzzle(source, "z-low", 1000)
	middle := testTrainingPuzzle(source, "m-middle", 1500)
	high := testTrainingPuzzle(source, "a-high", 2000)
	middle.Occurrence.Ordinal = 2
	high.Occurrence.Ordinal = 3
	sealAndActivate(t, beginGenerationImport(t, catalog, source), high, low, middle)

	got, err := catalog.FreePracticeCandidates(context.Background(), source.ID, nil, nil, nil, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprints := candidateFingerprints(got); !reflect.DeepEqual(fingerprints, []string{"z-low", "m-middle", "a-high"}) {
		t.Fatalf("unranged practice order = %q, want rating then fingerprint", fingerprints)
	}
	minimum, maximum := 1200, 2000
	got, err = catalog.FreePracticeCandidates(context.Background(), source.ID, &minimum, &maximum, nil, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprints := candidateFingerprints(got); !reflect.DeepEqual(fingerprints, []string{"m-middle", "a-high"}) {
		t.Fatalf("ranged practice order = %q, want rating then fingerprint", fingerprints)
	}
}

func TestReaderMembershipPlansUsePhysicalPrimaryKeys(t *testing.T) {
	_, store := openTestGenerationalCatalog(t)
	alpha := testSource("alpha-plan", "test", "/alpha-plan")
	beta := testSource("beta-plan", "test", "/beta-plan")
	alphaPuzzle := testTrainingPuzzle(alpha, "shared-plan", 1500, "fork", "pin")
	betaPuzzle := testTrainingPuzzle(beta, "shared-plan", 1400, "fork")
	alphaGeneration := seedActiveReaderGeneration(t, store, alpha, alphaPuzzle)
	seedActiveReaderGeneration(t, store, beta, betaPuzzle)

	plans := map[string][]string{
		"rated preferred-source de-dup": catalogQueryPlanDetails(t, store.Reader, `
			SELECT rated.fingerprint, rated.rating_key
			FROM occurrence_ratings rated
			WHERE rated.generation_id = ?
			  AND rated.rating_key BETWEEN ? AND ?
			  AND rated.rating_key <> ?
			  AND NOT EXISTS (
			    SELECT 1
			    FROM source_heads preferred_head
			    CROSS JOIN source_generations preferred_generation
			    CROSS JOIN puzzle_occurrences preferred_occurrence
			    CROSS JOIN occurrence_ratings preferred_rating
			    WHERE preferred_generation.source_id = preferred_head.source_id
			      AND preferred_generation.generation_id = preferred_head.generation_id
			      AND preferred_occurrence.generation_id = preferred_head.generation_id
			      AND preferred_occurrence.fingerprint = rated.fingerprint
			      AND preferred_rating.generation_id = preferred_occurrence.generation_id
			      AND preferred_rating.rating_key = preferred_occurrence.rating
			      AND preferred_rating.fingerprint = preferred_occurrence.fingerprint
			      AND preferred_rating.rating_key BETWEEN ? AND ?
			      AND preferred_head.source_id < ?
			      AND preferred_generation.status = 'sealed'
			  )
			ORDER BY rated.rating_key, rated.fingerprint LIMIT ?`,
			alphaGeneration, 1000, 2000, nullPuzzleRatingKey, 1000, 2000, alpha.ID, 25),
		"free practice unranged": catalogQueryPlanDetails(t, store.Reader, `
			SELECT rated.fingerprint
			FROM occurrence_ratings rated
			WHERE rated.generation_id = ?
			ORDER BY rated.rating_key, rated.fingerprint LIMIT ?`, alphaGeneration, 25),
		"free practice ranged": catalogQueryPlanDetails(t, store.Reader, `
			SELECT rated.fingerprint
			FROM occurrence_ratings rated
			WHERE rated.generation_id = ?
			  AND rated.rating_key BETWEEN ? AND ?
			  AND rated.rating_key <> ?
			ORDER BY rated.rating_key, rated.fingerprint LIMIT ?`,
			alphaGeneration, 1000, 2000, nullPuzzleRatingKey, 25),
		"theme membership count": catalogQueryPlanDetails(t, store.Reader, `
			SELECT COUNT(*)
			FROM occurrence_themes theme
			WHERE theme.generation_id = ? AND theme.theme IN (?, ?)`,
			alphaGeneration, "fork", "pin"),
		"membership-first themes": catalogQueryPlanDetails(t, store.Reader, `
			SELECT DISTINCT theme.fingerprint
			FROM occurrence_themes theme
			JOIN puzzle_occurrences occurrence
			  ON occurrence.fingerprint = theme.fingerprint
			 AND occurrence.generation_id = theme.generation_id
			JOIN puzzle_cores core ON core.fingerprint = theme.fingerprint
			WHERE theme.generation_id = ? AND theme.theme IN (?, ?)
			ORDER BY occurrence.rating, theme.fingerprint LIMIT ?`,
			alphaGeneration, "fork", "pin", 25),
		"rating-first themes": catalogQueryPlanDetails(t, store.Reader, `
			SELECT rated.fingerprint
			FROM occurrence_ratings rated
			JOIN puzzle_cores core ON core.fingerprint = rated.fingerprint
			WHERE rated.generation_id = ?
			  AND rated.rating_key BETWEEN ? AND ?
			  AND rated.rating_key <> ?
			  AND EXISTS (
			    SELECT 1 FROM occurrence_themes theme
			    WHERE theme.generation_id = rated.generation_id
			      AND theme.theme IN (?, ?)
			      AND theme.fingerprint = rated.fingerprint
			  )
			ORDER BY rated.rating_key, rated.fingerprint LIMIT ?`,
			alphaGeneration, 1000, 2000, nullPuzzleRatingKey, "fork", "pin", 25),
		"core orphan cleanup": catalogQueryPlanDetails(t, store.Reader, `
			SELECT core.rowid
			FROM puzzle_cores core
			WHERE NOT EXISTS (
			  SELECT 1 FROM puzzle_occurrences occurrence
			  WHERE occurrence.fingerprint = core.fingerprint
			)
			ORDER BY core.fingerprint LIMIT ?`, 1000),
	}
	assertQueryPlanContains(t, plans["rated preferred-source de-dup"], "preferred_occurrence USING PRIMARY KEY (fingerprint=? AND generation_id=?)")
	assertQueryPlanContains(t, plans["rated preferred-source de-dup"], "rated USING PRIMARY KEY (generation_id=? AND rating_key>? AND rating_key<?)")
	assertQueryPlanContains(t, plans["free practice unranged"], "rated USING PRIMARY KEY (generation_id=?)")
	assertQueryPlanContains(t, plans["free practice ranged"], "rated USING PRIMARY KEY (generation_id=? AND rating_key>? AND rating_key<?)")
	assertQueryPlanContains(t, plans["theme membership count"], "theme USING PRIMARY KEY (generation_id=? AND theme=?)")
	assertQueryPlanNotContains(t, plans["theme membership count"], "USE TEMP B-TREE")
	assertQueryPlanContains(t, plans["membership-first themes"], "theme USING PRIMARY KEY (generation_id=? AND theme=?)")
	assertQueryPlanContains(t, plans["membership-first themes"], "occurrence USING PRIMARY KEY (fingerprint=? AND generation_id=?)")
	assertQueryPlanContains(t, plans["rating-first themes"], "rated USING PRIMARY KEY (generation_id=? AND rating_key>? AND rating_key<?)")
	assertQueryPlanContains(t, plans["rating-first themes"], "theme EXISTS USING PRIMARY KEY (generation_id=? AND theme=? AND fingerprint=?)")
	assertQueryPlanContains(t, plans["core orphan cleanup"], "occurrence USING PRIMARY KEY (fingerprint=?)")
}

func candidateFingerprints(puzzles []TrainingPuzzle) []string {
	fingerprints := make([]string, len(puzzles))
	for index, puzzle := range puzzles {
		fingerprints[index] = puzzle.Core.Fingerprint
	}
	return fingerprints
}

func catalogQueryPlanDetails(t *testing.T, db *sql.DB, query string, args ...any) []string {
	t.Helper()
	rows, err := db.Query(`EXPLAIN QUERY PLAN `+query, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var details []string
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatal(err)
		}
		details = append(details, detail)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return details
}

func assertQueryPlanContains(t *testing.T, details []string, want string) {
	t.Helper()
	for _, detail := range details {
		if strings.Contains(detail, want) {
			return
		}
	}
	t.Fatalf("query plan = %s; want fragment %q", strings.Join(details, " | "), want)
}

func assertQueryPlanNotContains(t *testing.T, details []string, unwanted string) {
	t.Helper()
	for _, detail := range details {
		if strings.Contains(detail, unwanted) {
			t.Fatalf("query plan = %s; unwanted fragment %q", strings.Join(details, " | "), unwanted)
		}
	}
}
