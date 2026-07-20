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
	completedActivityIDs []string,
	now time.Time,
) error {
	return writeLessonProgressTx(
		ctx, tx, courseID, lessonID, completedActivityIDs,
		len(completedActivityIDs), len(completedActivityIDs), now, true,
	)
}

func (s *UserStore) RecordActivityProgress(ctx context.Context, update ActivityProgressUpdate) error {
	if s == nil || s.db == nil {
		return errors.New("opening user store is required")
	}
	if err := validateActivityProgressUpdate(update); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stored, alreadyCompleted, err := storedLessonProgressTx(ctx, tx, update.CourseID, update.LessonID)
	if err != nil {
		return err
	}
	completedSet := make(map[string]struct{}, len(stored)+1)
	for _, activityID := range stored {
		completedSet[activityID] = struct{}{}
	}
	completedSet[update.CompletedActivityID] = struct{}{}
	ordered := orderCompletedActivityIDs(update.RequiredActivityIDs, stored, completedSet)
	completedCurrent := 0
	for _, activityID := range update.RequiredActivityIDs {
		if _, completed := completedSet[activityID]; completed {
			completedCurrent++
		}
	}
	completed := alreadyCompleted || completedCurrent == len(update.RequiredActivityIDs)
	if err := writeLessonProgressTx(
		ctx, tx, update.CourseID, update.LessonID, ordered,
		completedCurrent, len(update.RequiredActivityIDs), update.Now, completed,
	); err != nil {
		return err
	}
	return tx.Commit()
}

func storedLessonProgressTx(
	ctx context.Context,
	tx *sql.Tx,
	courseID string,
	lessonID string,
) ([]string, bool, error) {
	var encoded string
	var completedAt sql.NullInt64
	err := tx.QueryRowContext(
		ctx,
		`SELECT completed_activity_ids_json, completed_at
		 FROM opening_lesson_progress WHERE course_id = ? AND lesson_id = ?`,
		courseID,
		lessonID,
	).Scan(&encoded, &completedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return []string{}, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var stored []string
	if err := json.Unmarshal([]byte(encoded), &stored); err != nil {
		return nil, false, fmt.Errorf("decode opening lesson progress: %w", err)
	}
	return stored, completedAt.Valid, nil
}

func orderCompletedActivityIDs(required, stored []string, completedSet map[string]struct{}) []string {
	ordered := make([]string, 0, len(completedSet))
	added := make(map[string]struct{}, len(completedSet))
	for _, activityID := range required {
		if _, completed := completedSet[activityID]; completed {
			ordered = append(ordered, activityID)
			added[activityID] = struct{}{}
		}
	}
	for _, activityID := range stored {
		if _, exists := added[activityID]; !exists {
			ordered = append(ordered, activityID)
			added[activityID] = struct{}{}
		}
	}
	return ordered
}

func writeLessonProgressTx(
	ctx context.Context,
	tx *sql.Tx,
	courseID string,
	lessonID string,
	completedActivityIDs []string,
	completedActivities int,
	totalActivities int,
	now time.Time,
	completed bool,
) error {
	encoded, err := json.Marshal(completedActivityIDs)
	if err != nil {
		return err
	}
	var completedAt any
	if completed {
		completedAt = now.UnixMilli()
	}
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO opening_lesson_progress(
		   course_id, lesson_id, completed_activity_ids_json, completed_activities,
		   total_activities, completed_at, updated_at
		 ) VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(course_id, lesson_id) DO UPDATE SET
		   completed_activity_ids_json = excluded.completed_activity_ids_json,
		   completed_activities = excluded.completed_activities,
		   total_activities = excluded.total_activities,
		   completed_at = COALESCE(opening_lesson_progress.completed_at, excluded.completed_at),
		   updated_at = excluded.updated_at`,
		courseID,
		lessonID,
		string(encoded),
		completedActivities,
		totalActivities,
		completedAt,
		now.UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("record opening lesson progress: %w", err)
	}
	return nil
}

func (s *UserStore) LessonProgress(
	ctx context.Context,
	courseID string,
	lessonID string,
	activeActivityIDs []string,
) (LessonProgress, error) {
	if err := validateCourseKey(courseID); err != nil {
		return LessonProgress{}, err
	}
	if strings.TrimSpace(lessonID) == "" {
		return LessonProgress{}, errors.New("opening lesson ID is required")
	}
	if err := validateActivityIDs(activeActivityIDs); err != nil {
		return LessonProgress{}, err
	}
	result := LessonProgress{
		CourseID: courseID, LessonID: lessonID,
		CompletedActivityIDs: []string{}, TotalActivities: len(activeActivityIDs),
	}
	var encoded string
	var completedAt sql.NullInt64
	err := s.db.QueryRowContext(
		ctx,
		`SELECT completed_activity_ids_json, completed_at
		 FROM opening_lesson_progress WHERE course_id = ? AND lesson_id = ?`,
		courseID,
		lessonID,
	).Scan(&encoded, &completedAt)
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
	for _, activityID := range stored {
		storedSet[activityID] = struct{}{}
	}
	for _, activityID := range activeActivityIDs {
		if _, completed := storedSet[activityID]; completed {
			result.CompletedActivityIDs = append(result.CompletedActivityIDs, activityID)
		}
	}
	result.CompletedActivities = len(result.CompletedActivityIDs)
	result.Completed = completedAt.Valid
	return result, nil
}

func validateActivityProgressUpdate(update ActivityProgressUpdate) error {
	if err := validateCourseKey(update.CourseID); err != nil {
		return err
	}
	if strings.TrimSpace(update.LessonID) == "" {
		return errors.New("opening lesson ID is required")
	}
	if strings.TrimSpace(update.CompletedActivityID) == "" {
		return errors.New("completed opening activity ID is required")
	}
	if err := validateActivityIDs(update.RequiredActivityIDs); err != nil {
		return err
	}
	found := false
	for _, activityID := range update.RequiredActivityIDs {
		found = found || activityID == update.CompletedActivityID
	}
	if !found {
		return fmt.Errorf("completed opening activity ID %q is not required", update.CompletedActivityID)
	}
	if update.Now.IsZero() {
		return errors.New("opening activity progress time is required")
	}
	return nil
}

func validateActivityIDs(activityIDs []string) error {
	if len(activityIDs) == 0 {
		return errors.New("opening lesson must contain at least one required activity")
	}
	seen := make(map[string]struct{}, len(activityIDs))
	for _, activityID := range activityIDs {
		if strings.TrimSpace(activityID) == "" {
			return errors.New("opening lesson activity ID is required")
		}
		if _, duplicate := seen[activityID]; duplicate {
			return fmt.Errorf("opening lesson activity ID %q is duplicated", activityID)
		}
		seen[activityID] = struct{}{}
	}
	return nil
}
