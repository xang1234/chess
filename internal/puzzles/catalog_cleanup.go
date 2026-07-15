package puzzles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func (c *GenerationalSQLiteCatalog) RecoverStartup(ctx context.Context) error {
	if _, err := c.writeDB.ExecContext(
		ctx,
		`UPDATE source_generations
		 SET status = 'abandoned'
		 WHERE status = 'building'`,
	); err != nil {
		return fmt.Errorf("abandon interrupted puzzle generations: %w", err)
	}
	return nil
}

func (c *GenerationalSQLiteCatalog) CleanupBatch(ctx context.Context, limit int) (bool, error) {
	if limit <= 0 {
		return false, errors.New("cleanup limit must be positive")
	}
	tx, err := c.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin puzzle catalogue cleanup: %w", err)
	}
	defer tx.Rollback()

	budget := int64(limit)
	phases := []struct {
		name      string
		statement string
	}{
		{
			name: "occurrence themes",
			statement: `DELETE FROM occurrence_themes
				WHERE rowid IN (
				  SELECT theme.rowid
				  FROM occurrence_themes theme
				  JOIN source_generations generation
				    ON generation.generation_id = theme.generation_id
				  WHERE generation.status IN ('abandoned', 'sealed')
				    AND NOT EXISTS (
				      SELECT 1 FROM source_heads head
				      WHERE head.generation_id = generation.generation_id
				    )
				  ORDER BY theme.generation_id, theme.fingerprint, theme.theme
				  LIMIT ?
				)`,
		},
		{
			name: "puzzle occurrences",
			statement: `DELETE FROM puzzle_occurrences
				WHERE rowid IN (
				  SELECT occurrence.rowid
				  FROM puzzle_occurrences occurrence
				  JOIN source_generations generation
				    ON generation.generation_id = occurrence.generation_id
				  WHERE generation.status IN ('abandoned', 'sealed')
				    AND NOT EXISTS (
				      SELECT 1 FROM source_heads head
				      WHERE head.generation_id = generation.generation_id
				    )
				    AND NOT EXISTS (
				      SELECT 1 FROM occurrence_themes theme
				      WHERE theme.generation_id = occurrence.generation_id
				        AND theme.fingerprint = occurrence.fingerprint
				    )
				  ORDER BY occurrence.generation_id, occurrence.fingerprint
				  LIMIT ?
				)`,
		},
		{
			name: "source generations",
			statement: `DELETE FROM source_generations
				WHERE rowid IN (
				  SELECT generation.rowid
				  FROM source_generations generation
				  WHERE generation.status IN ('abandoned', 'sealed')
				    AND NOT EXISTS (
				      SELECT 1 FROM source_heads head
				      WHERE head.generation_id = generation.generation_id
				    )
				    AND NOT EXISTS (
				      SELECT 1 FROM puzzle_occurrences occurrence
				      WHERE occurrence.generation_id = generation.generation_id
				    )
				  ORDER BY generation.generation_id
				  LIMIT ?
				)`,
		},
		{
			name: "puzzle sources",
			statement: `DELETE FROM sources
				WHERE rowid IN (
				  SELECT source.rowid
				  FROM sources source
				  WHERE NOT EXISTS (
				    SELECT 1 FROM source_generations generation
				    WHERE generation.source_id = source.source_id
				  )
				    AND NOT EXISTS (
				      SELECT 1 FROM source_heads head
				      WHERE head.source_id = source.source_id
				    )
				  ORDER BY source.source_id
				  LIMIT ?
				)`,
		},
		{
			name: "puzzle cores",
			statement: `DELETE FROM puzzle_cores
				WHERE rowid IN (
				  SELECT core.rowid
				  FROM puzzle_cores core
				  WHERE NOT EXISTS (
				    SELECT 1 FROM puzzle_occurrences occurrence
				    WHERE occurrence.fingerprint = core.fingerprint
				  )
				  ORDER BY core.fingerprint
				  LIMIT ?
				)`,
		},
	}
	for _, phase := range phases {
		if budget == 0 {
			break
		}
		deleted, err := executeCleanupPhase(ctx, tx, phase.name, phase.statement, budget)
		if err != nil {
			return false, err
		}
		budget -= deleted
	}

	more, err := cleanupRowsRemain(ctx, tx)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit puzzle catalogue cleanup: %w", err)
	}
	return more, nil
}

func executeCleanupPhase(
	ctx context.Context,
	tx *sql.Tx,
	name string,
	statement string,
	budget int64,
) (int64, error) {
	result, err := tx.ExecContext(ctx, statement, budget)
	if err != nil {
		return 0, fmt.Errorf("delete cleanup %s: %w", name, err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count cleanup %s: %w", name, err)
	}
	if deleted < 0 || deleted > budget {
		return 0, fmt.Errorf("cleanup %s deleted %d rows with budget %d", name, deleted, budget)
	}
	return deleted, nil
}

func cleanupRowsRemain(ctx context.Context, tx *sql.Tx) (bool, error) {
	var more bool
	if err := tx.QueryRowContext(
		ctx,
		`SELECT
		   EXISTS (
		     SELECT 1
		     FROM source_generations generation
		     WHERE generation.status IN ('abandoned', 'sealed')
		       AND NOT EXISTS (
		         SELECT 1 FROM source_heads head
		         WHERE head.generation_id = generation.generation_id
		       )
		   )
		   OR EXISTS (
		     SELECT 1
		     FROM sources source
		     WHERE NOT EXISTS (
		       SELECT 1 FROM source_generations generation
		       WHERE generation.source_id = source.source_id
		     )
		       AND NOT EXISTS (
		         SELECT 1 FROM source_heads head
		         WHERE head.source_id = source.source_id
		       )
		   )
		   OR EXISTS (
		     SELECT 1
		     FROM puzzle_cores core
		     WHERE NOT EXISTS (
		       SELECT 1 FROM puzzle_occurrences occurrence
		       WHERE occurrence.fingerprint = core.fingerprint
		     )
		   )`,
	).Scan(&more); err != nil {
		return false, fmt.Errorf("check remaining puzzle cleanup rows: %w", err)
	}
	return more, nil
}
