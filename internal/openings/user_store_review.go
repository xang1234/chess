package openings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"chess-trainer/internal/spacedreview"
)

func (s *UserStore) CompletePrompt(
	ctx context.Context,
	completion PromptCompletion,
	now time.Time,
) error {
	if err := validatePromptCompletion(completion, now); err != nil {
		return err
	}
	stateJSON, err := encodeSessionState(completion.Session.State)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	attempt := completion.Attempt
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO opening_attempts(
		   attempt_id, session_id, course_id, prompt_id, semantic_fingerprint,
		   started_at, completed_at, outcome, incorrect_moves,
		   alternatives_tried, hints_used, revealed
		 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		attempt.AttemptID,
		completion.Session.ID,
		completion.Session.CourseID,
		attempt.PromptID,
		completion.SemanticFingerprint,
		attempt.StartedAt.UnixMilli(),
		now.UnixMilli(),
		completion.Outcome,
		attempt.IncorrectMoves,
		attempt.AlternativesTried,
		attempt.HintsUsed,
		attempt.Revealed,
	); err != nil {
		return fmt.Errorf("insert opening attempt: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO opening_prompt_progress(
		   course_id, prompt_id, semantic_fingerprint, last_outcome, updated_at
		 ) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(course_id, prompt_id) DO UPDATE SET
		   semantic_fingerprint = excluded.semantic_fingerprint,
		   last_outcome = excluded.last_outcome,
		   updated_at = excluded.updated_at`,
		completion.Session.CourseID,
		attempt.PromptID,
		completion.SemanticFingerprint,
		completion.Outcome,
		now.UnixMilli(),
	); err != nil {
		return fmt.Errorf("update opening prompt progress: %w", err)
	}
	if err := upsertOpeningReview(ctx, tx, completion, now); err != nil {
		return err
	}
	if len(completion.CompletedStepIDs) != 0 {
		if err := upsertLessonProgress(
			ctx, tx, completion.Session.CourseID, completion.Session.LessonID,
			completion.CompletedStepIDs, now,
		); err != nil {
			return err
		}
	}
	result, err := tx.ExecContext(
		ctx,
		`UPDATE opening_sessions
		 SET status = ?, depth = ?, step_index = ?, state_json = ?, updated_at = ?
		 WHERE session_id = ? AND course_id = ? AND generation_id = ?
		   AND lesson_id = ? AND mode = ?`,
		completion.Session.Status,
		completion.Session.Depth,
		completion.Session.StepIndex,
		stateJSON,
		now.UnixMilli(),
		completion.Session.ID,
		completion.Session.CourseID,
		completion.Session.GenerationID,
		completion.Session.LessonID,
		completion.Session.Mode,
	)
	if err != nil {
		return fmt.Errorf("update opening session after prompt: %w", err)
	}
	if err := requireOneSessionRow(result, completion.Session.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func upsertOpeningReview(
	ctx context.Context,
	tx *sql.Tx,
	completion PromptCompletion,
	now time.Time,
) error {
	state := spacedreview.State{IntervalIndex: -1}
	var storedFingerprint string
	err := tx.QueryRowContext(
		ctx,
		`SELECT semantic_fingerprint, interval_index, successful_reviews
		 FROM opening_review_state WHERE course_id = ? AND prompt_id = ?`,
		completion.Session.CourseID,
		completion.Attempt.PromptID,
	).Scan(&storedFingerprint, &state.IntervalIndex, &state.SuccessfulReviews)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("read opening review: %w", err)
	}
	if errors.Is(err, sql.ErrNoRows) || storedFingerprint != completion.SemanticFingerprint {
		state = spacedreview.State{IntervalIndex: -1}
	}
	scheduled := spacedreview.Next(now, state, completion.Outcome)
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO opening_review_state(
		   course_id, prompt_id, semantic_fingerprint, due_at, interval_index,
		   successful_reviews, last_outcome, status
		 ) VALUES (?, ?, ?, ?, ?, ?, ?, 'active')
		 ON CONFLICT(course_id, prompt_id) DO UPDATE SET
		   semantic_fingerprint = excluded.semantic_fingerprint,
		   due_at = excluded.due_at,
		   interval_index = excluded.interval_index,
		   successful_reviews = excluded.successful_reviews,
		   last_outcome = excluded.last_outcome,
		   status = 'active'`,
		completion.Session.CourseID,
		completion.Attempt.PromptID,
		completion.SemanticFingerprint,
		scheduled.DueAt.UnixMilli(),
		scheduled.State.IntervalIndex,
		scheduled.State.SuccessfulReviews,
		completion.Outcome,
	); err != nil {
		return fmt.Errorf("update opening review: %w", err)
	}
	return nil
}

func (s *UserStore) Review(
	ctx context.Context,
	courseID string,
	promptID string,
) (ReviewState, error) {
	var state ReviewState
	var dueAt int64
	err := s.db.QueryRowContext(
		ctx,
		`SELECT course_id, prompt_id, semantic_fingerprint, due_at,
		        interval_index, successful_reviews, last_outcome, status
		 FROM opening_review_state WHERE course_id = ? AND prompt_id = ?`,
		courseID,
		promptID,
	).Scan(
		&state.CourseID,
		&state.PromptID,
		&state.SemanticFingerprint,
		&dueAt,
		&state.IntervalIndex,
		&state.SuccessfulReviews,
		&state.LastOutcome,
		&state.Status,
	)
	state.DueAt = time.UnixMilli(dueAt).UTC()
	return state, err
}

func (s *UserStore) DueReviews(
	ctx context.Context,
	courseID string,
	now time.Time,
	limit int,
) ([]ReviewState, error) {
	if limit <= 0 {
		return []ReviewState{}, nil
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT course_id, prompt_id, semantic_fingerprint, due_at,
		        interval_index, successful_reviews, last_outcome, status
		 FROM opening_review_state
		 WHERE course_id = ? AND status = 'active' AND due_at <= ?
		 ORDER BY due_at, prompt_id LIMIT ?`,
		courseID,
		now.UnixMilli(),
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	states := make([]ReviewState, 0, limit)
	for rows.Next() {
		var state ReviewState
		var dueAt int64
		if err := rows.Scan(
			&state.CourseID,
			&state.PromptID,
			&state.SemanticFingerprint,
			&dueAt,
			&state.IntervalIndex,
			&state.SuccessfulReviews,
			&state.LastOutcome,
			&state.Status,
		); err != nil {
			return nil, err
		}
		state.DueAt = time.UnixMilli(dueAt).UTC()
		states = append(states, state)
	}
	return states, rows.Err()
}

func (s *UserStore) ProtectedGenerationIDs(
	ctx context.Context,
) (map[string]struct{}, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT DISTINCT generation_id FROM opening_sessions
		 WHERE status IN ('active','paused','restart_required')
		 ORDER BY generation_id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	protected := map[string]struct{}{}
	for rows.Next() {
		var generationID string
		if err := rows.Scan(&generationID); err != nil {
			return nil, err
		}
		protected[generationID] = struct{}{}
	}
	return protected, rows.Err()
}

func validatePromptCompletion(completion PromptCompletion, now time.Time) error {
	if err := validateStoredSession(completion.Session); err != nil {
		return err
	}
	state := completion.Attempt
	if strings.TrimSpace(state.AttemptID) == "" {
		return errors.New("opening attempt ID is required")
	}
	if strings.TrimSpace(state.PromptID) == "" {
		return errors.New("opening prompt ID is required")
	}
	if strings.TrimSpace(completion.SemanticFingerprint) == "" {
		return errors.New("opening prompt fingerprint is required")
	}
	if state.StartedAt.IsZero() || now.IsZero() || now.Before(state.StartedAt) {
		return errors.New("valid opening attempt times are required")
	}
	if state.IncorrectMoves < 0 || state.AlternativesTried < 0 || state.HintsUsed < 0 {
		return errors.New("opening attempt metrics cannot be negative")
	}
	switch completion.Outcome {
	case ReviewClean, ReviewMissed, ReviewHinted, ReviewRevealed:
	default:
		return fmt.Errorf("invalid opening review outcome %q", completion.Outcome)
	}
	if len(completion.CompletedStepIDs) != 0 {
		if completion.Session.Mode != OpeningModeLesson {
			return errors.New("review sessions cannot complete a lesson")
		}
		if err := validateStepIDs(completion.CompletedStepIDs); err != nil {
			return err
		}
	}
	return nil
}
