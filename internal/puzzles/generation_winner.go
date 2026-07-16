package puzzles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const generationAppendSchema = "generation_append"

const generationWinnerSchemaSQL = `CREATE TABLE winner_rows (
	fingerprint TEXT PRIMARY KEY,
	displayed_fen TEXT NOT NULL,
	solver TEXT NOT NULL,
	solution_json TEXT NOT NULL,
	solution_plies INTEGER NOT NULL,
	external_id TEXT,
	source_fen TEXT,
	prelude_uci TEXT,
	rating INTEGER,
	popularity INTEGER,
	play_count INTEGER,
	source_url TEXT,
	attribution TEXT,
	metadata_json TEXT NOT NULL,
	themes_json TEXT NOT NULL,
	ordinal INTEGER NOT NULL,
	copies INTEGER NOT NULL,
	core_conflict INTEGER NOT NULL CHECK (core_conflict IN (0, 1))
) WITHOUT ROWID;
CREATE INDEX winner_core_conflicts
ON winner_rows(core_conflict)
WHERE core_conflict = 1`

const generationWinnerBuildSQL = `INSERT INTO winner_rows(
	fingerprint, displayed_fen, solver, solution_json, solution_plies,
	external_id, source_fen, prelude_uci, rating, popularity, play_count,
	source_url, attribution, metadata_json, themes_json, ordinal,
	copies, core_conflict
) SELECT fingerprint, displayed_fen, solver, solution_json, solution_plies,
	       external_id, source_fen, prelude_uci, rating, popularity, play_count,
	       source_url, attribution, metadata_json, themes_json, ordinal,
	       copies, core_conflict
	FROM (
	  SELECT staged.fingerprint, staged.displayed_fen, staged.solver,
	         staged.solution_json, staged.solution_plies, staged.external_id,
	         staged.source_fen, staged.prelude_uci, staged.rating,
	         staged.popularity, staged.play_count, staged.source_url,
	         staged.attribution, staged.metadata_json, staged.themes_json,
	         staged.ordinal,
	         ROW_NUMBER() OVER fingerprint_rows AS winner_rank,
	         COUNT(*) OVER fingerprint_rows AS copies,
	         CASE WHEN MIN(staged.displayed_fen) OVER fingerprint_rows
	                        IS NOT MAX(staged.displayed_fen) OVER fingerprint_rows
	                    OR MIN(staged.solver) OVER fingerprint_rows
	                        IS NOT MAX(staged.solver) OVER fingerprint_rows
	                    OR MIN(staged.solution_json) OVER fingerprint_rows
	                        IS NOT MAX(staged.solution_json) OVER fingerprint_rows
	                    OR MIN(staged.solution_plies) OVER fingerprint_rows
	                        IS NOT MAX(staged.solution_plies) OVER fingerprint_rows
	              THEN 1 ELSE 0 END AS core_conflict
	  FROM generation_append.staged_rows AS staged
	  WINDOW fingerprint_rows AS (
	    PARTITION BY staged.fingerprint
	    ORDER BY staged.row_id DESC
	    ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING
	  )
	) AS ranked
	WHERE winner_rank = 1`

const generationWinnerCoreConflictQuery = `SELECT fingerprint
	FROM winner_rows
	WHERE core_conflict = 1
	LIMIT 1`

type generationWinner struct {
	db   *sql.DB
	path string
}

func createGenerationWinner(
	ctx context.Context,
	mainDB *sql.DB,
	generationID string,
) (*generationWinner, error) {
	db, winnerPath, err := createGenerationArtifact(
		ctx,
		mainDB,
		"winner",
		generationID,
		generationWinnerSchemaSQL,
	)
	if err != nil {
		return nil, err
	}
	return &generationWinner{db: db, path: winnerPath}, nil
}

func (s *sqliteGenerationImport) compactToWinner(
	ctx context.Context,
) (result *generationWinner, stagedRows, winners int64, returnErr error) {
	if s.stage == nil {
		return nil, 0, 0, errors.New("generation append stage is unavailable")
	}
	winner, err := createGenerationWinner(ctx, s.catalog.writeDB, s.generationID)
	if err != nil {
		return nil, 0, 0, err
	}
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.Join(returnErr, winner.closeAndRemove())
	}()
	if err := s.stage.close(); err != nil {
		return nil, 0, 0, fmt.Errorf("close generation append stage before compaction: %w", err)
	}
	if _, err := winner.db.ExecContext(
		ctx,
		`ATTACH DATABASE ? AS `+generationAppendSchema,
		s.stage.path,
	); err != nil {
		return nil, 0, 0, fmt.Errorf("attach generation append stage: %w", err)
	}
	attached := true
	defer func() {
		if !attached || winner.db == nil {
			return
		}
		if _, err := winner.db.ExecContext(
			context.WithoutCancel(ctx),
			`DETACH DATABASE `+generationAppendSchema,
		); err != nil {
			returnErr = errors.Join(
				returnErr,
				fmt.Errorf("detach generation append stage: %w", err),
			)
		}
	}()
	if _, err := winner.db.ExecContext(ctx, generationWinnerBuildSQL); err != nil {
		return nil, 0, 0, fmt.Errorf("build generation winners: %w", err)
	}
	if err := validateGenerationWinnerCoreConflicts(ctx, winner.db); err != nil {
		return nil, 0, 0, err
	}
	if err := winner.db.QueryRowContext(
		ctx,
		`SELECT COALESCE(SUM(copies), 0), COUNT(*) FROM winner_rows`,
	).Scan(&stagedRows, &winners); err != nil {
		return nil, 0, 0, fmt.Errorf("count generation winners: %w", err)
	}
	if _, err := winner.db.ExecContext(ctx, `DETACH DATABASE `+generationAppendSchema); err != nil {
		return nil, 0, 0, fmt.Errorf("detach compacted generation append stage: %w", err)
	}
	attached = false
	if err := winner.close(); err != nil {
		return nil, 0, 0, fmt.Errorf("close compacted generation winner: %w", err)
	}
	if err := s.stage.remove(); err != nil {
		return nil, 0, 0, fmt.Errorf("remove consumed generation append stage: %w", err)
	}
	s.stage = nil
	return winner, stagedRows, winners, nil
}

func validateGenerationWinnerCoreConflicts(ctx context.Context, db *sql.DB) error {
	var fingerprint string
	err := db.QueryRowContext(ctx, generationWinnerCoreConflictQuery).Scan(&fingerprint)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("validate generation winner cores: %w", err)
	}
	return fmt.Errorf(
		"%w: fingerprint %q maps to different stable content",
		ErrCatalogCorrupt,
		fingerprint,
	)
}

func (w *generationWinner) close() error {
	if w == nil || w.db == nil {
		return nil
	}
	err := w.db.Close()
	w.db = nil
	return err
}

func (w *generationWinner) remove() error {
	if w == nil {
		return nil
	}
	return removeGenerationArtifact(w.path)
}

func (w *generationWinner) closeAndRemove() error {
	if w == nil {
		return nil
	}
	return errors.Join(w.close(), w.remove())
}
