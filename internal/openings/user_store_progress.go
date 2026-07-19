package openings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

func upsertLessonProgress(
	ctx context.Context,
	tx *sql.Tx,
	courseID string,
	lessonID string,
	completedStepIDs []string,
	now time.Time,
) error {
	encoded, err := json.Marshal(completedStepIDs)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO opening_lesson_progress(
		   course_id, lesson_id, completed_step_ids_json, completed_steps,
		   total_steps, completed_at, updated_at
		 ) VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(course_id, lesson_id) DO UPDATE SET
		   completed_step_ids_json = excluded.completed_step_ids_json,
		   completed_steps = excluded.completed_steps,
		   total_steps = excluded.total_steps,
		   completed_at = excluded.completed_at,
		   updated_at = excluded.updated_at`,
		courseID,
		lessonID,
		string(encoded),
		len(completedStepIDs),
		len(completedStepIDs),
		now.UnixMilli(),
		now.UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("complete opening lesson: %w", err)
	}
	return nil
}

func (s *UserStore) LessonProgress(
	ctx context.Context,
	courseID string,
	lessonID string,
	activeStepIDs []string,
) (LessonProgress, error) {
	if err := validateStepIDs(activeStepIDs); err != nil {
		return LessonProgress{}, err
	}
	result := LessonProgress{
		CourseID: courseID, LessonID: lessonID,
		CompletedStepIDs: []string{}, TotalSteps: len(activeStepIDs),
	}
	var encoded string
	var cachedTotal int
	var completedAt sql.NullInt64
	err := s.db.QueryRowContext(
		ctx,
		`SELECT completed_step_ids_json, total_steps, completed_at
		 FROM opening_lesson_progress WHERE course_id = ? AND lesson_id = ?`,
		courseID,
		lessonID,
	).Scan(&encoded, &cachedTotal, &completedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return result, nil
	}
	if err != nil {
		return LessonProgress{}, err
	}
	var stored []string
	if err := json.Unmarshal([]byte(encoded), &stored); err != nil {
		return LessonProgress{}, fmt.Errorf("decode opening lesson progress: %w", err)
	}
	storedSet := make(map[string]struct{}, len(stored))
	for _, stepID := range stored {
		storedSet[stepID] = struct{}{}
	}
	for _, stepID := range activeStepIDs {
		if _, completed := storedSet[stepID]; completed {
			result.CompletedStepIDs = append(result.CompletedStepIDs, stepID)
		}
	}
	result.CompletedSteps = len(result.CompletedStepIDs)
	result.Completed = completedAt.Valid && cachedTotal == result.TotalSteps &&
		result.CompletedSteps == result.TotalSteps
	return result, nil
}

func validateStepIDs(stepIDs []string) error {
	if len(stepIDs) == 0 {
		return errors.New("opening lesson must contain at least one step")
	}
	seen := make(map[string]struct{}, len(stepIDs))
	for _, stepID := range stepIDs {
		if strings.TrimSpace(stepID) == "" {
			return errors.New("opening lesson step ID is required")
		}
		if _, duplicate := seen[stepID]; duplicate {
			return fmt.Errorf("opening lesson step ID %q is duplicated", stepID)
		}
		seen[stepID] = struct{}{}
	}
	return nil
}
