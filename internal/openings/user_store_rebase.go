package openings

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *UserStore) RebaseSession(
	ctx context.Context,
	previousGenerationID string,
	session StoredSession,
	now time.Time,
) error {
	if s == nil || s.db == nil {
		return errors.New("opening user store is required")
	}
	if strings.TrimSpace(previousGenerationID) == "" {
		return errors.New("previous course generation ID is required")
	}
	if err := validateStoredSession(session); err != nil {
		return err
	}
	if now.IsZero() {
		return errors.New("opening session rebase time is required")
	}
	stateJSON, err := encodeSessionState(session.State)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE opening_sessions
		 SET generation_id = ?, lesson_id = ?, status = ?, depth = ?,
		     step_index = ?, state_json = ?, updated_at = ?
		 WHERE session_id = ? AND course_id = ? AND generation_id = ? AND mode = ?`,
		session.GenerationID,
		session.LessonID,
		session.Status,
		session.Depth,
		session.StepIndex,
		stateJSON,
		now.UnixMilli(),
		session.ID,
		session.CourseID,
		previousGenerationID,
		session.Mode,
	)
	if err != nil {
		return fmt.Errorf("rebase opening session: %w", err)
	}
	return requireOneSessionRow(result, session.ID)
}

func (s *UserStore) ReconcileReviews(
	ctx context.Context,
	courseID string,
	activeFingerprints map[string]string,
) error {
	if s == nil || s.db == nil {
		return errors.New("opening user store is required")
	}
	if err := validateCourseKey(courseID); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
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
	return tx.Commit()
}
