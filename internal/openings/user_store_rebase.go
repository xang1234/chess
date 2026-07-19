package openings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *UserStore) ApplyCourseRevision(ctx context.Context, revision CourseRevision) error {
	if s == nil || s.db == nil {
		return errors.New("opening user store is required")
	}
	if err := validateCourseKey(revision.CourseID); err != nil {
		return err
	}
	if revision.Now.IsZero() {
		return errors.New("opening course revision time is required")
	}
	for promptID, fingerprint := range revision.PromptFingerprints {
		if strings.TrimSpace(promptID) == "" || strings.TrimSpace(fingerprint) == "" {
			return errors.New("opening course revision fingerprints must be non-empty")
		}
	}
	if revision.SessionRebase != nil {
		if strings.TrimSpace(revision.SessionRebase.PreviousGenerationID) == "" {
			return errors.New("previous course generation ID is required")
		}
		if err := validateStoredSession(revision.SessionRebase.Session); err != nil {
			return err
		}
		if revision.SessionRebase.Session.CourseID != revision.CourseID {
			return errors.New("opening session rebase course does not match revision course")
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if revision.SessionRebase != nil {
		if err := rebaseSessionTx(ctx, tx, *revision.SessionRebase, revision.Now); err != nil {
			return err
		}
	}
	if err := reconcileReviewsTx(
		ctx, tx, revision.CourseID, revision.PromptFingerprints,
	); err != nil {
		return fmt.Errorf("reconcile opening reviews: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit opening course revision: %w", err)
	}
	return nil
}

func rebaseSessionTx(
	ctx context.Context,
	tx *sql.Tx,
	rebase SessionRebase,
	now time.Time,
) error {
	stateJSON, err := encodeSessionState(rebase.Session.State)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(
		ctx,
		`UPDATE opening_sessions
		 SET generation_id = ?, lesson_id = ?, status = ?, depth = ?,
		     step_index = ?, state_json = ?, updated_at = ?
		 WHERE session_id = ? AND course_id = ? AND generation_id = ? AND mode = ?`,
		rebase.Session.GenerationID,
		rebase.Session.LessonID,
		rebase.Session.Status,
		rebase.Session.Depth,
		rebase.Session.StepIndex,
		stateJSON,
		now.UnixMilli(),
		rebase.Session.ID,
		rebase.Session.CourseID,
		rebase.PreviousGenerationID,
		rebase.Session.Mode,
	)
	if err != nil {
		return fmt.Errorf("rebase opening session: %w", err)
	}
	return requireOneSessionRow(result, rebase.Session.ID)
}

func reconcileReviewsTx(
	ctx context.Context,
	tx *sql.Tx,
	courseID string,
	activeFingerprints map[string]string,
) error {
	rows, err := tx.QueryContext(
		ctx,
		`SELECT prompt_id, semantic_fingerprint
		 FROM opening_review_state WHERE course_id = ? AND status = 'active'`,
		courseID,
	)
	if err != nil {
		return err
	}
	archive := []string{}
	for rows.Next() {
		var promptID, fingerprint string
		if err := rows.Scan(&promptID, &fingerprint); err != nil {
			rows.Close()
			return err
		}
		if active, exists := activeFingerprints[promptID]; !exists || active != fingerprint {
			archive = append(archive, promptID)
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, promptID := range archive {
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE opening_review_state SET status = 'archived'
			 WHERE course_id = ? AND prompt_id = ? AND status = 'active'`,
			courseID,
			promptID,
		); err != nil {
			return err
		}
	}
	return nil
}
