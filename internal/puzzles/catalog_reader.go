package puzzles

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"chess-trainer/internal/domain"
)

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
	query := strings.Builder{}
	query.WriteString(`WITH ranked AS (
		SELECT occurrence.fingerprint,
		       head.source_id,
		       occurrence.rating,
		       ROW_NUMBER() OVER (
		         PARTITION BY occurrence.fingerprint
		         ORDER BY head.source_id
		       ) AS source_rank
		FROM source_heads head
		JOIN source_generations generation
		  ON generation.source_id = head.source_id
		 AND generation.generation_id = head.generation_id
		JOIN puzzle_occurrences occurrence
		  ON occurrence.generation_id = head.generation_id
		WHERE generation.status = 'sealed'
		  AND occurrence.rating BETWEEN ? AND ?`)
	args := []any{minimum, maximum}
	if len(excluded) > 0 {
		query.WriteString(` AND occurrence.fingerprint NOT IN (`)
		query.WriteString(generationPlaceholders(len(excluded)))
		query.WriteString(`)`)
		for _, fingerprint := range excluded {
			args = append(args, fingerprint)
		}
	}
	query.WriteString(`)
		SELECT fingerprint, source_id
		FROM ranked
		WHERE source_rank = 1
		ORDER BY rating, fingerprint, source_id
		LIMIT ?`)
	args = append(args, limit)
	return c.readCandidates(ctx, query.String(), args...)
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
	query := strings.Builder{}
	query.WriteString(`SELECT occurrence.fingerprint, head.source_id
		FROM source_heads head
		JOIN source_generations generation
		  ON generation.source_id = head.source_id
		 AND generation.generation_id = head.generation_id
		JOIN puzzle_occurrences occurrence
		  ON occurrence.generation_id = head.generation_id
		JOIN puzzle_cores core ON core.fingerprint = occurrence.fingerprint
		WHERE head.source_id = ?
		  AND generation.status = 'sealed'`)
	args := []any{sourceID}
	if minimum != nil {
		query.WriteString(` AND occurrence.rating >= ?`)
		args = append(args, *minimum)
	}
	if maximum != nil {
		query.WriteString(` AND occurrence.rating <= ?`)
		args = append(args, *maximum)
	}
	if len(themes) > 0 {
		query.WriteString(` AND EXISTS (
			SELECT 1 FROM occurrence_themes theme
			WHERE theme.generation_id = occurrence.generation_id
			  AND theme.fingerprint = occurrence.fingerprint
			  AND theme.theme IN (`)
		query.WriteString(generationPlaceholders(len(themes)))
		query.WriteString(`))`)
		for _, theme := range themes {
			args = append(args, theme)
		}
	}
	if maximumSolutionPlies != nil {
		query.WriteString(` AND core.solution_plies <= ?`)
		args = append(args, *maximumSolutionPlies)
	}
	query.WriteString(` ORDER BY occurrence.fingerprint LIMIT ?`)
	args = append(args, limit)
	return c.readCandidates(ctx, query.String(), args...)
}

func generationPlaceholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", count), ",")
}

func (c *SQLiteCatalog) readCandidates(
	ctx context.Context,
	query string,
	args ...any,
) ([]TrainingPuzzle, error) {
	tx, err := c.readDB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin puzzle candidate read: %w", err)
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query puzzle candidates: %w", err)
	}
	keys := make([]PuzzleKey, 0)
	for rows.Next() {
		var key PuzzleKey
		if err := rows.Scan(&key.Fingerprint, &key.SourceID); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan puzzle candidate: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close puzzle candidates: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate puzzle candidates: %w", err)
	}

	puzzles := make([]TrainingPuzzle, 0, len(keys))
	for _, key := range keys {
		puzzle, err := readActiveTrainingPuzzle(ctx, tx, key)
		if err != nil {
			return nil, err
		}
		puzzles = append(puzzles, puzzle)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("finish puzzle candidate read: %w", err)
	}
	return puzzles, nil
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
		        MIN(occurrence.rating),
		        MAX(occurrence.rating),
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

func (c *SQLiteCatalog) ActiveThemes(ctx context.Context) ([]string, error) {
	tx, err := c.readDB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin active theme read: %w", err)
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(
		ctx,
		`SELECT DISTINCT theme.theme
		 FROM source_heads head
		 JOIN source_generations generation
		   ON generation.source_id = head.source_id
		  AND generation.generation_id = head.generation_id
		 JOIN occurrence_themes theme
		   ON theme.generation_id = head.generation_id
		 WHERE generation.status = 'sealed'
		 ORDER BY theme.theme`,
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
	var solver, solutionJSON, sourceKind, metadataJSON string
	var externalID, sourceFEN, preludeUCI, sourceURL, attribution sql.NullString
	var rating, popularity, playCount sql.NullInt64
	err := tx.QueryRowContext(
		ctx,
		`SELECT
		   core.displayed_fen, core.solver, core.solution_json, core.solution_plies,
		   source.kind, occurrence.external_id, occurrence.source_fen,
		   occurrence.prelude_uci, occurrence.rating, occurrence.popularity,
		   occurrence.play_count, occurrence.source_url, occurrence.attribution,
		   occurrence.metadata_json, occurrence.ordinal
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

	rows, err := tx.QueryContext(
		ctx,
		`SELECT theme
		 FROM occurrence_themes theme
		 JOIN source_heads head ON head.generation_id = theme.generation_id
		 WHERE head.source_id = ? AND theme.fingerprint = ?
		 ORDER BY theme.theme`,
		key.SourceID,
		key.Fingerprint,
	)
	if err != nil {
		return TrainingPuzzle{}, fmt.Errorf("read puzzle occurrence themes: %w", err)
	}
	for rows.Next() {
		var theme string
		if err := rows.Scan(&theme); err != nil {
			rows.Close()
			return TrainingPuzzle{}, fmt.Errorf("scan puzzle occurrence theme: %w", err)
		}
		puzzle.Occurrence.Themes = append(puzzle.Occurrence.Themes, theme)
	}
	if err := rows.Close(); err != nil {
		return TrainingPuzzle{}, fmt.Errorf("close puzzle occurrence themes: %w", err)
	}
	if err := rows.Err(); err != nil {
		return TrainingPuzzle{}, fmt.Errorf("iterate puzzle occurrence themes: %w", err)
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
