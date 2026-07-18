package puzzles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"chess-trainer/internal/domain"
)

const themeMembershipFirstMaximum = 1_000

// nullPuzzleRatingKey keeps unrated occurrences in the ordered membership
// table without conflating the canonical nullable rating on the occurrence.
const nullPuzzleRatingKey int64 = -1 << 63

func (c *SQLiteCatalog) Get(ctx context.Context, key PuzzleKey) (TrainingPuzzle, error) {
	tx, err := c.readDB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return TrainingPuzzle{}, fmt.Errorf("begin puzzle read: %w", err)
	}
	defer tx.Rollback()
	puzzle, err := readActiveTrainingPuzzle(ctx, tx, key)
	if err != nil {
		return TrainingPuzzle{}, err
	}
	if err := tx.Commit(); err != nil {
		return TrainingPuzzle{}, fmt.Errorf("finish puzzle read: %w", err)
	}
	return puzzle, nil
}

func (c *SQLiteCatalog) Resolve(
	ctx context.Context,
	fingerprint string,
	preferredSourceID string,
) (TrainingPuzzle, error) {
	tx, err := c.readDB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return TrainingPuzzle{}, fmt.Errorf("begin puzzle resolve: %w", err)
	}
	defer tx.Rollback()
	var sourceID string
	if err := tx.QueryRowContext(
		ctx,
		`SELECT head.source_id
		 FROM source_heads head
		 JOIN source_generations generation
		   ON generation.source_id = head.source_id
		  AND generation.generation_id = head.generation_id
		 JOIN puzzle_occurrences occurrence
		   ON occurrence.generation_id = head.generation_id
		 WHERE occurrence.fingerprint = ?
		   AND generation.status = 'sealed'
		 ORDER BY CASE WHEN head.source_id = ? THEN 0 ELSE 1 END,
		          head.source_id
		 LIMIT 1`,
		fingerprint,
		preferredSourceID,
	).Scan(&sourceID); err != nil {
		return TrainingPuzzle{}, err
	}
	puzzle, err := readActiveTrainingPuzzle(ctx, tx, PuzzleKey{
		Fingerprint: fingerprint,
		SourceID:    sourceID,
	})
	if err != nil {
		return TrainingPuzzle{}, err
	}
	if err := tx.Commit(); err != nil {
		return TrainingPuzzle{}, fmt.Errorf("finish puzzle resolve: %w", err)
	}
	return puzzle, nil
}

func (c *SQLiteCatalog) RatedCandidates(
	ctx context.Context,
	minimum int,
	maximum int,
	excluded []string,
	limit int,
) ([]TrainingPuzzle, error) {
	if limit <= 0 {
		return []TrainingPuzzle{}, nil
	}
	tx, err := c.readDB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin rated puzzle candidate read: %w", err)
	}
	defer tx.Rollback()

	type activeHead struct {
		sourceID     string
		generationID string
	}
	headRows, err := tx.QueryContext(
		ctx,
		`SELECT head.source_id, head.generation_id
		 FROM source_heads head
		 JOIN source_generations generation
		   ON generation.source_id = head.source_id
		  AND generation.generation_id = head.generation_id
		 WHERE generation.status = 'sealed'
		 ORDER BY head.source_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("query active puzzle source heads: %w", err)
	}
	heads := make([]activeHead, 0)
	for headRows.Next() {
		var head activeHead
		if err := headRows.Scan(&head.sourceID, &head.generationID); err != nil {
			headRows.Close()
			return nil, fmt.Errorf("scan active puzzle source head: %w", err)
		}
		heads = append(heads, head)
	}
	if err := headRows.Close(); err != nil {
		return nil, fmt.Errorf("close active puzzle source heads: %w", err)
	}
	if err := headRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active puzzle source heads: %w", err)
	}

	queryForSide := func(descending bool) string {
		query := strings.Builder{}
		query.WriteString(`SELECT rated.fingerprint, rated.rating_key,
		       occurrence.popularity, occurrence.play_count
		FROM occurrence_ratings AS rated
		JOIN puzzle_occurrences AS occurrence
		  ON occurrence.generation_id = rated.generation_id
		 AND occurrence.fingerprint = rated.fingerprint
		WHERE rated.generation_id = ?
		  AND rated.rating_key BETWEEN ? AND ?
		  AND rated.rating_key <> ?`)
		if len(excluded) > 0 {
			query.WriteString(` AND rated.fingerprint NOT IN (`)
			query.WriteString(generationPlaceholders(len(excluded)))
			query.WriteString(`)`)
		}
		query.WriteString(` AND NOT EXISTS (
			SELECT 1
			FROM source_heads preferred_head
			CROSS JOIN source_generations preferred_generation
			CROSS JOIN puzzle_occurrences AS preferred_occurrence
			CROSS JOIN occurrence_ratings AS preferred_rating
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
		ORDER BY rated.rating_key `)
		if descending {
			query.WriteString(`DESC`)
		} else {
			query.WriteString(`ASC`)
		}
		query.WriteString(`,
		         occurrence.popularity IS NULL,
		         occurrence.popularity DESC,
		         occurrence.play_count IS NULL,
		         occurrence.play_count DESC,
		         rated.fingerprint
		LIMIT ?`)
		return query.String()
	}

	type ratedCandidate struct {
		key        PuzzleKey
		rating     int64
		popularity *int
		playCount  *int
	}
	candidates := make([]ratedCandidate, 0, len(heads)*limit*2)
	centerFloor := int64(minimum) + (int64(maximum)-int64(minimum))/2
	for _, head := range heads {
		for _, side := range []struct {
			minimum    int64
			maximum    int64
			descending bool
		}{
			{minimum: int64(minimum), maximum: min(int64(maximum), centerFloor), descending: true},
			{minimum: max(int64(minimum), centerFloor+1), maximum: int64(maximum)},
		} {
			if side.minimum > side.maximum {
				continue
			}
			args := []any{head.generationID, side.minimum, side.maximum, nullPuzzleRatingKey}
			for _, fingerprint := range excluded {
				args = append(args, fingerprint)
			}
			args = append(args, minimum, maximum, head.sourceID, limit)
			rows, err := tx.QueryContext(ctx, queryForSide(side.descending), args...)
			if err != nil {
				return nil, fmt.Errorf("query rated puzzle candidates for source %q: %w", head.sourceID, err)
			}
			for rows.Next() {
				candidate := ratedCandidate{key: PuzzleKey{SourceID: head.sourceID}}
				var popularity, playCount sql.NullInt64
				if err := rows.Scan(
					&candidate.key.Fingerprint,
					&candidate.rating,
					&popularity,
					&playCount,
				); err != nil {
					rows.Close()
					return nil, fmt.Errorf("scan rated puzzle candidate: %w", err)
				}
				candidate.popularity = generationIntPointer(popularity)
				candidate.playCount = generationIntPointer(playCount)
				candidates = append(candidates, candidate)
			}
			if err := rows.Close(); err != nil {
				return nil, fmt.Errorf("close rated puzzle candidates: %w", err)
			}
			if err := rows.Err(); err != nil {
				return nil, fmt.Errorf("iterate rated puzzle candidates: %w", err)
			}
		}
	}
	centerTwice := int64(minimum) + int64(maximum)
	sort.Slice(candidates, func(left, right int) bool {
		leftDistance := ratingDistanceTwice(candidates[left].rating, centerTwice)
		rightDistance := ratingDistanceTwice(candidates[right].rating, centerTwice)
		if leftDistance != rightDistance {
			return leftDistance < rightDistance
		}
		if comparison := compareOptionalQuality(candidates[left].popularity, candidates[right].popularity); comparison != 0 {
			return comparison > 0
		}
		if comparison := compareOptionalQuality(candidates[left].playCount, candidates[right].playCount); comparison != 0 {
			return comparison > 0
		}
		if candidates[left].key.SourceID != candidates[right].key.SourceID {
			return candidates[left].key.SourceID < candidates[right].key.SourceID
		}
		return candidates[left].key.Fingerprint < candidates[right].key.Fingerprint
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	result := make([]TrainingPuzzle, 0, len(candidates))
	for _, candidate := range candidates {
		puzzle, err := readActiveTrainingPuzzle(ctx, tx, candidate.key)
		if err != nil {
			return nil, err
		}
		result = append(result, puzzle)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("finish rated puzzle candidate read: %w", err)
	}
	return result, nil
}

func ratingDistanceTwice(rating, centerTwice int64) int64 {
	distance := 2*rating - centerTwice
	if distance < 0 {
		return -distance
	}
	return distance
}

func compareOptionalQuality(left, right *int) int {
	switch {
	case left == nil && right == nil:
		return 0
	case left == nil:
		return -1
	case right == nil:
		return 1
	case *left < *right:
		return -1
	case *left > *right:
		return 1
	default:
		return 0
	}
}

func (c *SQLiteCatalog) FreePracticeCandidates(
	ctx context.Context,
	sourceID string,
	minimum *int,
	maximum *int,
	themes []string,
	maximumSolutionPlies *int,
	limit int,
) ([]TrainingPuzzle, error) {
	if limit <= 0 {
		return []TrainingPuzzle{}, nil
	}
	return c.freePracticeCandidates(
		ctx,
		sourceID,
		minimum,
		maximum,
		themes,
		maximumSolutionPlies,
		limit,
	)
}

func (c *SQLiteCatalog) freePracticeCandidates(
	ctx context.Context,
	sourceID string,
	minimum *int,
	maximum *int,
	themes []string,
	maximumSolutionPlies *int,
	limit int,
) ([]TrainingPuzzle, error) {
	tx, err := c.readDB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin free-practice puzzle candidate read: %w", err)
	}
	defer tx.Rollback()

	var generationID string
	if err := tx.QueryRowContext(
		ctx,
		`SELECT head.generation_id
		 FROM source_heads head
		 JOIN source_generations generation
		   ON generation.source_id = head.source_id
		  AND generation.generation_id = head.generation_id
		 WHERE head.source_id = ? AND generation.status = 'sealed'`,
		sourceID,
	).Scan(&generationID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []TrainingPuzzle{}, nil
		}
		return nil, fmt.Errorf("read active generation for free-practice candidates: %w", err)
	}

	var membershipCount int64
	if len(themes) > 0 {
		// Overlap may count one fingerprint once per requested theme. That can
		// conservatively choose rating-first, but cannot change ANY semantics.
		countQuery := strings.Builder{}
		countQuery.WriteString(`SELECT COUNT(*)
			FROM occurrence_themes AS theme
			WHERE theme.generation_id = ? AND theme.theme IN (`)
		countQuery.WriteString(generationPlaceholders(len(themes)))
		countQuery.WriteString(`)`)
		countArgs := make([]any, 0, len(themes)+1)
		countArgs = append(countArgs, generationID)
		for _, theme := range themes {
			countArgs = append(countArgs, theme)
		}
		if err := tx.QueryRowContext(ctx, countQuery.String(), countArgs...).Scan(&membershipCount); err != nil {
			return nil, fmt.Errorf("count active theme memberships: %w", err)
		}
	}

	var query strings.Builder
	args := make([]any, 0, len(themes)+4)
	if len(themes) > 0 && membershipCount <= themeMembershipFirstMaximum {
		query.WriteString(`SELECT DISTINCT theme.fingerprint
			FROM occurrence_themes AS theme
			JOIN puzzle_occurrences AS occurrence
			  ON occurrence.fingerprint = theme.fingerprint
			 AND occurrence.generation_id = theme.generation_id
			JOIN puzzle_cores AS core ON core.fingerprint = theme.fingerprint
			WHERE theme.generation_id = ? AND theme.theme IN (`)
		query.WriteString(generationPlaceholders(len(themes)))
		query.WriteString(`)`)
		args = append(args, generationID)
		for _, theme := range themes {
			args = append(args, theme)
		}
		if minimum != nil {
			query.WriteString(` AND occurrence.rating >= ?`)
			args = append(args, *minimum)
		}
		if maximum != nil {
			query.WriteString(` AND occurrence.rating <= ?`)
			args = append(args, *maximum)
		}
		if maximumSolutionPlies != nil {
			query.WriteString(` AND core.solution_plies <= ?`)
			args = append(args, *maximumSolutionPlies)
		}
		query.WriteString(` ORDER BY occurrence.rating, theme.fingerprint LIMIT ?`)
	} else {
		query.WriteString(`SELECT rated.fingerprint
			FROM occurrence_ratings AS rated
			JOIN puzzle_cores AS core ON core.fingerprint = rated.fingerprint
			WHERE rated.generation_id = ?`)
		args = append(args, generationID)
		if minimum != nil {
			query.WriteString(` AND rated.rating_key >= ?`)
			args = append(args, *minimum)
		}
		if maximum != nil {
			query.WriteString(` AND rated.rating_key <= ?`)
			args = append(args, *maximum)
		}
		if minimum != nil || maximum != nil {
			query.WriteString(` AND rated.rating_key <> ?`)
			args = append(args, nullPuzzleRatingKey)
		}
		if len(themes) > 0 {
			query.WriteString(` AND EXISTS (
				SELECT 1 FROM occurrence_themes AS theme
				WHERE theme.generation_id = rated.generation_id
				  AND theme.theme IN (`)
			query.WriteString(generationPlaceholders(len(themes)))
			query.WriteString(`)
				  AND theme.fingerprint = rated.fingerprint
			)`)
			for _, theme := range themes {
				args = append(args, theme)
			}
		}
		if maximumSolutionPlies != nil {
			query.WriteString(` AND core.solution_plies <= ?`)
			args = append(args, *maximumSolutionPlies)
		}
		query.WriteString(` ORDER BY rated.rating_key, rated.fingerprint LIMIT ?`)
	}
	args = append(args, limit)
	puzzles, err := readSourceCandidates(ctx, tx, sourceID, query.String(), args...)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("finish free-practice puzzle candidate read: %w", err)
	}
	return puzzles, nil
}

func readSourceCandidates(
	ctx context.Context,
	tx *sql.Tx,
	sourceID string,
	query string,
	args ...any,
) ([]TrainingPuzzle, error) {
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query source puzzle candidates: %w", err)
	}
	fingerprints := make([]string, 0)
	for rows.Next() {
		var fingerprint string
		if err := rows.Scan(&fingerprint); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan source puzzle candidate: %w", err)
		}
		fingerprints = append(fingerprints, fingerprint)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close source puzzle candidates: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate source puzzle candidates: %w", err)
	}

	puzzles := make([]TrainingPuzzle, 0, len(fingerprints))
	for _, fingerprint := range fingerprints {
		puzzle, err := readActiveTrainingPuzzle(ctx, tx, PuzzleKey{
			Fingerprint: fingerprint,
			SourceID:    sourceID,
		})
		if err != nil {
			return nil, err
		}
		puzzles = append(puzzles, puzzle)
	}
	return puzzles, nil
}

func generationPlaceholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", count), ",")
}

func (c *SQLiteCatalog) ActiveSourceSummaries(ctx context.Context) ([]SourceSummary, error) {
	tx, err := c.readDB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin active source summary read: %w", err)
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(
		ctx,
		`SELECT head.source_id,
		        source.kind,
		        (SELECT rated.rating_key
		         FROM occurrence_ratings AS rated
		         WHERE rated.generation_id = head.generation_id
		           AND rated.rating_key <> ?
		         ORDER BY rated.rating_key, rated.fingerprint
		         LIMIT 1),
		        (SELECT rated.rating_key
		         FROM occurrence_ratings AS rated
		         WHERE rated.generation_id = head.generation_id
		           AND rated.rating_key <> ?
		         ORDER BY rated.rating_key DESC, rated.fingerprint DESC
		         LIMIT 1),
		        COALESCE(MAX(core.solution_plies), 0)
		 FROM source_heads head
		 JOIN source_generations generation
		   ON generation.source_id = head.source_id
		  AND generation.generation_id = head.generation_id
		 JOIN sources source ON source.source_id = head.source_id
		 LEFT JOIN puzzle_occurrences occurrence
		   ON occurrence.generation_id = head.generation_id
		 LEFT JOIN puzzle_cores core ON core.fingerprint = occurrence.fingerprint
		 WHERE generation.status = 'sealed'
		 GROUP BY head.source_id, source.kind
		 ORDER BY head.source_id`,
		nullPuzzleRatingKey,
		nullPuzzleRatingKey,
	)
	if err != nil {
		return nil, fmt.Errorf("query active source summaries: %w", err)
	}
	summaries := make([]SourceSummary, 0)
	for rows.Next() {
		var summary SourceSummary
		var minimum, maximum sql.NullInt64
		if err := rows.Scan(
			&summary.SourceID,
			&summary.Kind,
			&minimum,
			&maximum,
			&summary.MaximumSolutionPlies,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan active source summary: %w", err)
		}
		summary.MinimumRating = generationIntPointer(minimum)
		summary.MaximumRating = generationIntPointer(maximum)
		summaries = append(summaries, summary)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close active source summaries: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active source summaries: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("finish active source summary read: %w", err)
	}
	return summaries, nil
}

// LearnerRatingBounds returns the combined range of every currently active
// source with explicit ratings. A catalogue without active rated occurrences
// keeps the stable application defaults used before catalogue-backed bounds existed.
func (c *SQLiteCatalog) LearnerRatingBounds(ctx context.Context) (RatingBounds, error) {
	tx, err := c.readDB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return RatingBounds{}, fmt.Errorf("begin learner rating bounds read: %w", err)
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(
		ctx,
		`SELECT
		   source.kind,
		   (SELECT rated.rating_key
		    FROM occurrence_ratings AS rated
		    WHERE rated.generation_id = head.generation_id
		      AND rated.rating_key <> ?
		    ORDER BY rated.rating_key, rated.fingerprint
		    LIMIT 1),
		   (SELECT rated.rating_key
		    FROM occurrence_ratings AS rated
		    WHERE rated.generation_id = head.generation_id
		      AND rated.rating_key <> ?
		    ORDER BY rated.rating_key DESC, rated.fingerprint DESC
		    LIMIT 1)
		 FROM source_heads AS head
		 JOIN source_generations AS generation
		   ON generation.source_id = head.source_id
		  AND generation.generation_id = head.generation_id
		 JOIN sources AS source ON source.source_id = head.source_id
		 WHERE generation.status = 'sealed'`,
		nullPuzzleRatingKey,
		nullPuzzleRatingKey,
	)
	if err != nil {
		return RatingBounds{}, fmt.Errorf("query learner rating bounds: %w", err)
	}
	summaries := make([]SourceSummary, 0)
	for rows.Next() {
		var kind string
		var minimum, maximum sql.NullInt64
		if err := rows.Scan(&kind, &minimum, &maximum); err != nil {
			rows.Close()
			return RatingBounds{}, fmt.Errorf("scan learner rating bounds: %w", err)
		}
		summaries = append(summaries, SourceSummary{
			Kind:          kind,
			MinimumRating: generationIntPointer(minimum),
			MaximumRating: generationIntPointer(maximum),
		})
	}
	if err := rows.Close(); err != nil {
		return RatingBounds{}, fmt.Errorf("close learner rating bounds: %w", err)
	}
	if err := rows.Err(); err != nil {
		return RatingBounds{}, fmt.Errorf("iterate learner rating bounds: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return RatingBounds{}, fmt.Errorf("finish learner rating bounds read: %w", err)
	}
	return LearnerRatingBoundsFromSourceSummaries(summaries), nil
}

func (c *SQLiteCatalog) ActiveThemes(ctx context.Context) ([]string, error) {
	tx, err := c.readDB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin active theme read: %w", err)
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(
		ctx,
		`SELECT DISTINCT facet.theme
		 FROM source_heads head
		 JOIN source_generations generation
		   ON generation.source_id = head.source_id
		  AND generation.generation_id = head.generation_id
		 JOIN generation_themes facet
		   ON facet.generation_id = head.generation_id
		 WHERE generation.status = 'sealed'
		 ORDER BY facet.theme`,
	)
	if err != nil {
		return nil, fmt.Errorf("query active themes: %w", err)
	}
	themes := make([]string, 0)
	for rows.Next() {
		var theme string
		if err := rows.Scan(&theme); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan active theme: %w", err)
		}
		themes = append(themes, theme)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close active themes: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active themes: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("finish active theme read: %w", err)
	}
	return themes, nil
}

func readActiveTrainingPuzzle(
	ctx context.Context,
	tx *sql.Tx,
	key PuzzleKey,
) (TrainingPuzzle, error) {
	var puzzle TrainingPuzzle
	var solver, solutionJSON, sourceKind, metadataJSON, themesJSON string
	var externalID, sourceFEN, preludeUCI, sourceURL, attribution sql.NullString
	var rating, popularity, playCount sql.NullInt64
	err := tx.QueryRowContext(
		ctx,
		`SELECT
		   core.displayed_fen, core.solver, core.solution_json, core.solution_plies,
		   source.kind, occurrence.external_id, occurrence.source_fen,
		   occurrence.prelude_uci, occurrence.rating, occurrence.popularity,
		   occurrence.play_count, occurrence.source_url, occurrence.attribution,
		   occurrence.metadata_json, occurrence.themes_json, occurrence.ordinal
		 FROM source_heads head
		 JOIN source_generations generation
		   ON generation.source_id = head.source_id
		  AND generation.generation_id = head.generation_id
		 JOIN sources source ON source.source_id = head.source_id
		 JOIN puzzle_occurrences occurrence
		   ON occurrence.generation_id = head.generation_id
		 JOIN puzzle_cores core ON core.fingerprint = occurrence.fingerprint
		 WHERE head.source_id = ?
		   AND occurrence.fingerprint = ?
		   AND generation.status = 'sealed'`,
		key.SourceID,
		key.Fingerprint,
	).Scan(
		&puzzle.Core.DisplayedFEN,
		&solver,
		&solutionJSON,
		&puzzle.Core.SolutionPlies,
		&sourceKind,
		&externalID,
		&sourceFEN,
		&preludeUCI,
		&rating,
		&popularity,
		&playCount,
		&sourceURL,
		&attribution,
		&metadataJSON,
		&themesJSON,
		&puzzle.Occurrence.Ordinal,
	)
	if err != nil {
		return TrainingPuzzle{}, err
	}
	puzzle.Core.Fingerprint = key.Fingerprint
	puzzle.Core.Solver = domain.Color(solver)
	if err := json.Unmarshal([]byte(solutionJSON), &puzzle.Core.Solution); err != nil {
		return TrainingPuzzle{}, fmt.Errorf("decode puzzle solution %q: %w", key.Fingerprint, err)
	}
	puzzle.Occurrence.SourceID = key.SourceID
	puzzle.Occurrence.SourceKind = sourceKind
	puzzle.Occurrence.ExternalID = externalID.String
	puzzle.Occurrence.SourceFEN = sourceFEN.String
	puzzle.Occurrence.PreludeUCI = preludeUCI.String
	puzzle.Occurrence.Rating = generationIntPointer(rating)
	puzzle.Occurrence.Popularity = generationIntPointer(popularity)
	puzzle.Occurrence.PlayCount = generationIntPointer(playCount)
	puzzle.Occurrence.URL = sourceURL.String
	puzzle.Occurrence.Attribution = attribution.String
	if err := json.Unmarshal([]byte(metadataJSON), &puzzle.Occurrence.Metadata); err != nil {
		return TrainingPuzzle{}, fmt.Errorf("decode puzzle occurrence metadata %q: %w", key.Fingerprint, err)
	}
	if err := json.Unmarshal([]byte(themesJSON), &puzzle.Occurrence.Themes); err != nil {
		return TrainingPuzzle{}, fmt.Errorf("decode puzzle occurrence themes %q: %w", key.Fingerprint, err)
	}
	return puzzle, nil
}

func generationIntPointer(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	converted := int(value.Int64)
	return &converted
}
