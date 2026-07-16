package puzzles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const generationWinnerSchema = "generation_winner"

const generationCoreMaterializationSQL = `INSERT OR IGNORE INTO puzzle_cores(
	  fingerprint, displayed_fen, solver, solution_json, solution_plies
	) SELECT winner.fingerprint, winner.displayed_fen, winner.solver,
	         winner.solution_json, winner.solution_plies
	  FROM generation_winner.winner_rows AS winner
	  ORDER BY winner.fingerprint`

const generationOccurrenceMaterializationSQL = `INSERT INTO puzzle_occurrences(
	  generation_id, fingerprint, external_id, source_fen, prelude_uci,
	  rating, popularity, play_count, source_url, attribution,
	  metadata_json, themes_json, ordinal
	) SELECT ?, winner.fingerprint, winner.external_id, winner.source_fen,
	         winner.prelude_uci, winner.rating, winner.popularity,
	         winner.play_count, winner.source_url, winner.attribution,
	         winner.metadata_json, winner.themes_json, winner.ordinal
	  FROM generation_winner.winner_rows AS winner
	  ORDER BY winner.fingerprint`

const generationRatingMaterializationSQL = `INSERT INTO occurrence_ratings(
	  generation_id, rating_key, fingerprint
	) SELECT ?, COALESCE(winner.rating, ?), winner.fingerprint
	  FROM generation_winner.winner_rows AS winner
	  ORDER BY COALESCE(winner.rating, ?), winner.fingerprint`

const generationThemeMaterializationSQL = `INSERT INTO occurrence_themes(
	  generation_id, theme, fingerprint
	) SELECT ?, CAST(theme.value AS TEXT), winner.fingerprint
	  FROM generation_winner.winner_rows AS winner
	  CROSS JOIN json_each(winner.themes_json) AS theme
	  ORDER BY CAST(theme.value AS TEXT), winner.fingerprint`

const generationRatingMaterializationAuditSQL = `SELECT
	  generation_id, rating_key, fingerprint
	FROM (
	  SELECT ? AS generation_id,
	         COALESCE(winner.rating, ?) AS rating_key,
	         winner.fingerprint AS fingerprint,
	         1 AS delta
	  FROM generation_winner.winner_rows AS winner
	  UNION ALL
	  SELECT rated.generation_id, rated.rating_key, rated.fingerprint, -1 AS delta
	  FROM occurrence_ratings AS rated
	  WHERE rated.generation_id = ?
	)
	GROUP BY generation_id, rating_key, fingerprint
	HAVING SUM(delta) <> 0
	LIMIT 1`

const generationThemeMaterializationAuditSQL = `SELECT
	  generation_id, theme, fingerprint
	FROM (
	  SELECT ? AS generation_id,
	         CAST(theme.value AS TEXT) AS theme,
	         winner.fingerprint AS fingerprint,
	         1 AS delta
	  FROM generation_winner.winner_rows AS winner
	  CROSS JOIN json_each(winner.themes_json) AS theme
	  UNION ALL
	  SELECT themed.generation_id, themed.theme, themed.fingerprint, -1 AS delta
	  FROM occurrence_themes AS themed
	  WHERE themed.generation_id = ?
	)
	GROUP BY generation_id, theme, fingerprint
	HAVING SUM(delta) <> 0
	LIMIT 1`

// TODO(task6-reimport-scale): add a million-row reimport gate and replace this
// core-first lookup plan if both the canonical core table and stage are large.
// It deliberately makes the first-import preflight proportional to the small
// existing catalogue, but a full reimport can still random-probe winner rows.
const generationExistingCoreConflictQuery = `SELECT winner.fingerprint
	FROM puzzle_cores AS core
	CROSS JOIN generation_winner.winner_rows AS winner
	WHERE winner.fingerprint = core.fingerprint
	  AND (
	    core.displayed_fen IS NOT winner.displayed_fen
	    OR core.solver IS NOT winner.solver
	    OR core.solution_json IS NOT winner.solution_json
	    OR core.solution_plies IS NOT winner.solution_plies
	  )
	LIMIT 1`

func (s *sqliteGenerationImport) materialize(ctx context.Context) (returnErr error) {
	defer func() {
		if returnErr == nil || s.stage == nil {
			return
		}
		cleanupErr := s.stage.closeAndRemove()
		s.stage = nil
		if cleanupErr != nil {
			returnErr = errors.Join(
				returnErr,
				fmt.Errorf("clean failed generation stage: %w", cleanupErr),
			)
		}
	}()
	winner, stagedRows, winners, err := s.compactToWinner(ctx)
	if err != nil {
		return err
	}
	s.report.Accepted = winners
	s.report.Duplicates = stagedRows - winners
	defer func() {
		removeErr := winner.closeAndRemove()
		if removeErr != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("remove generation winner: %w", removeErr))
		}
	}()
	return s.materializeWinner(ctx, winner)
}

func (s *sqliteGenerationImport) materializeWinner(
	ctx context.Context,
	winner *generationWinner,
) (returnErr error) {
	if _, err := s.catalog.writeDB.ExecContext(
		ctx,
		`ATTACH DATABASE ? AS `+generationWinnerSchema,
		winner.path,
	); err != nil {
		return fmt.Errorf("attach generation winner: %w", err)
	}
	attached := true
	defer func() {
		if !attached {
			return
		}
		if _, err := s.catalog.writeDB.ExecContext(
			context.WithoutCancel(ctx),
			`DETACH DATABASE `+generationWinnerSchema,
		); err != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("detach generation winner: %w", err))
		}
	}()
	if err := s.validateExistingCores(ctx); err != nil {
		return err
	}

	phases := []struct {
		name               string
		statement          string
		suspendForeignKeys bool
		args               []any
	}{
		{
			name:      "cores",
			statement: generationCoreMaterializationSQL,
		},
		{
			name:      "occurrences",
			statement: generationOccurrenceMaterializationSQL,
			args:      []any{s.generationID},
		},
		{
			name:               "ratings",
			statement:          generationRatingMaterializationSQL,
			suspendForeignKeys: true,
			args:               []any{s.generationID, nullPuzzleRatingKey, nullPuzzleRatingKey},
		},
		{
			name:               "themes",
			statement:          generationThemeMaterializationSQL,
			suspendForeignKeys: true,
			args:               []any{s.generationID},
		},
	}
	for _, phase := range phases {
		if err := s.runMaterializationPhase(
			ctx,
			phase.name,
			phase.statement,
			phase.suspendForeignKeys,
			phase.args...,
		); err != nil {
			return err
		}
	}
	if err := s.validateMaterializedRows(ctx); err != nil {
		return err
	}
	if _, err := s.catalog.writeDB.ExecContext(ctx, `DETACH DATABASE `+generationWinnerSchema); err != nil {
		return fmt.Errorf("detach materialized generation winner: %w", err)
	}
	attached = false
	return nil
}

func (s *sqliteGenerationImport) runMaterializationPhase(
	ctx context.Context,
	name string,
	statement string,
	suspendForeignKeys bool,
	args ...any,
) error {
	if suspendForeignKeys {
		return s.runMaterializationPhaseWithoutForeignKeyProbes(
			ctx,
			name,
			statement,
			args...,
		)
	}
	return s.runMaterializationPhaseWithAutoCheckpoint(
		ctx,
		name,
		statement,
		setPuzzleWriterAutoCheckpoint,
		args...,
	)
}

func (s *sqliteGenerationImport) runMaterializationPhaseWithoutForeignKeyProbes(
	ctx context.Context,
	name string,
	statement string,
	args ...any,
) (returnErr error) {
	restored := false
	defer func() {
		if restored {
			return
		}
		if err := setPuzzleWriterForeignKeys(
			context.WithoutCancel(ctx),
			s.catalog.writeDB,
			true,
		); err != nil {
			returnErr = errors.Join(
				returnErr,
				fmt.Errorf("restore foreign keys after generation %s: %w", name, err),
			)
		}
	}()
	if err := setPuzzleWriterForeignKeys(ctx, s.catalog.writeDB, false); err != nil {
		return fmt.Errorf("suspend foreign keys for generation %s: %w", name, err)
	}
	if err := s.runMaterializationPhaseWithAutoCheckpoint(
		ctx,
		name,
		statement,
		setPuzzleWriterAutoCheckpoint,
		args...,
	); err != nil {
		return err
	}
	if err := setPuzzleWriterForeignKeys(ctx, s.catalog.writeDB, true); err != nil {
		return fmt.Errorf("restore foreign keys after generation %s: %w", name, err)
	}
	restored = true
	return nil
}

type puzzleAutoCheckpointSetter func(context.Context, *sql.DB, int) error

func (s *sqliteGenerationImport) runMaterializationPhaseWithAutoCheckpoint(
	ctx context.Context,
	name string,
	statement string,
	setAutoCheckpoint puzzleAutoCheckpointSetter,
	args ...any,
) (returnErr error) {
	restored := false
	defer func() {
		if restored {
			return
		}
		if err := setAutoCheckpoint(
			context.WithoutCancel(ctx),
			s.catalog.writeDB,
			16_384,
		); err != nil {
			returnErr = errors.Join(
				returnErr,
				fmt.Errorf("restore auto-checkpoint after generation %s: %w", name, err),
			)
		}
	}()
	if err := setAutoCheckpoint(ctx, s.catalog.writeDB, 0); err != nil {
		return fmt.Errorf("disable auto-checkpoint for generation %s: %w", name, err)
	}
	if _, err := s.catalog.writeDB.ExecContext(ctx, statement, args...); err != nil {
		return fmt.Errorf("materialize generation %s: %w", name, err)
	}
	var busy, logPages, checkpointed int
	if err := s.catalog.writeDB.QueryRowContext(
		ctx,
		`PRAGMA main.wal_checkpoint(TRUNCATE)`,
	).Scan(&busy, &logPages, &checkpointed); err != nil {
		return fmt.Errorf("checkpoint materialized generation %s: %w", name, err)
	}
	if busy != 0 {
		return fmt.Errorf(
			"checkpoint materialized generation %s remained busy (log=%d checkpointed=%d)",
			name,
			logPages,
			checkpointed,
		)
	}
	if err := setAutoCheckpoint(ctx, s.catalog.writeDB, 16_384); err != nil {
		return fmt.Errorf("restore auto-checkpoint after generation %s: %w", name, err)
	}
	restored = true
	return nil
}

func setPuzzleWriterAutoCheckpoint(ctx context.Context, db *sql.DB, pages int) error {
	if _, err := db.ExecContext(
		ctx,
		fmt.Sprintf("PRAGMA wal_autocheckpoint=%d", pages),
	); err != nil {
		return err
	}
	var actual int
	if err := db.QueryRowContext(ctx, `PRAGMA wal_autocheckpoint`).Scan(&actual); err != nil {
		return err
	}
	if actual != pages {
		return fmt.Errorf("wal_autocheckpoint=%d, want %d", actual, pages)
	}
	return nil
}

func setPuzzleWriterForeignKeys(ctx context.Context, db *sql.DB, enabled bool) error {
	value := 0
	if enabled {
		value = 1
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf("PRAGMA foreign_keys=%d", value)); err != nil {
		return err
	}
	var actual int
	if err := db.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&actual); err != nil {
		return err
	}
	if actual != value {
		return fmt.Errorf("foreign_keys=%d, want %d", actual, value)
	}
	return nil
}

func (s *sqliteGenerationImport) validateMaterializedRows(ctx context.Context) error {
	var generationID, fingerprint string
	var ratingKey int64
	err := s.catalog.writeDB.QueryRowContext(
		ctx,
		generationRatingMaterializationAuditSQL,
		s.generationID,
		nullPuzzleRatingKey,
		s.generationID,
	).Scan(&generationID, &ratingKey, &fingerprint)
	if err == nil {
		return fmt.Errorf(
			"%w: rating materialization audit differs at generation %q rating %d fingerprint %q",
			ErrCatalogCorrupt,
			generationID,
			ratingKey,
			fingerprint,
		)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("run rating materialization audit: %w", err)
	}

	var theme string
	err = s.catalog.writeDB.QueryRowContext(
		ctx,
		generationThemeMaterializationAuditSQL,
		s.generationID,
		s.generationID,
	).Scan(&generationID, &theme, &fingerprint)
	if err == nil {
		return fmt.Errorf(
			"%w: theme materialization audit differs at generation %q theme %q fingerprint %q",
			ErrCatalogCorrupt,
			generationID,
			theme,
			fingerprint,
		)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("run theme materialization audit: %w", err)
	}
	return s.validateMaterializedOccurrenceForeignKeys(ctx)
}

func (s *sqliteGenerationImport) validateMaterializedOccurrenceForeignKeys(
	ctx context.Context,
) error {
	rows, err := s.catalog.writeDB.QueryContext(
		ctx,
		`PRAGMA main.foreign_key_check(puzzle_occurrences)`,
	)
	if err != nil {
		return fmt.Errorf("run materialized occurrence foreign key check: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return fmt.Errorf("read materialized occurrence foreign key check: %w", err)
		}
		return nil
	}
	var table, parent string
	var rowID sql.NullInt64
	var foreignKeyID int
	if err := rows.Scan(&table, &rowID, &parent, &foreignKeyID); err != nil {
		return fmt.Errorf("read materialized occurrence foreign key check violation: %w", err)
	}
	return fmt.Errorf(
		"%w: foreign key check failed for table %q rowid %v parent %q constraint %d",
		ErrCatalogCorrupt,
		table,
		rowID,
		parent,
		foreignKeyID,
	)
}

func (s *sqliteGenerationImport) validateExistingCores(ctx context.Context) error {
	var fingerprint string
	err := s.catalog.writeDB.QueryRowContext(ctx, generationExistingCoreConflictQuery).Scan(&fingerprint)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("validate existing puzzle cores: %w", err)
	}
	return fmt.Errorf(
		"%w: fingerprint %q maps to different stable content",
		ErrCatalogCorrupt,
		fingerprint,
	)
}
