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

func (s *UserStore) Journey(ctx context.Context, courseID string, fallback Depth) (CourseJourney, error) {
	if s == nil || s.db == nil {
		return CourseJourney{}, errors.New("opening user store is required")
	}
	if err := validateCourseKey(courseID); err != nil {
		return CourseJourney{}, err
	}
	if _, ok := depthRank(fallback); !ok {
		return CourseJourney{}, fmt.Errorf("invalid fallback opening depth %q", fallback)
	}
	journey := CourseJourney{CourseID: courseID, Depth: fallback, PathLessonIDs: []string{}}
	var pathJSON string
	var createdAt, updatedAt int64
	err := s.db.QueryRowContext(
		ctx,
		`SELECT course_id, depth, current_lesson_id, current_activity_id,
		        path_lesson_ids_json, last_recommended_lesson_id, active_session_id,
		        created_at, updated_at
		 FROM opening_course_journeys WHERE course_id = ?`,
		courseID,
	).Scan(
		&journey.CourseID,
		&journey.Depth,
		&journey.CurrentLessonID,
		&journey.CurrentActivityID,
		&pathJSON,
		&journey.LastRecommendedLessonID,
		&journey.ActiveSessionID,
		&createdAt,
		&updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return journey, nil
	}
	if err != nil {
		return CourseJourney{}, err
	}
	if err := json.Unmarshal([]byte(pathJSON), &journey.PathLessonIDs); err != nil {
		return CourseJourney{}, fmt.Errorf("decode opening course journey path: %w", err)
	}
	journey.CreatedAt = time.UnixMilli(createdAt).UTC()
	journey.UpdatedAt = time.UnixMilli(updatedAt).UTC()
	return journey, nil
}

func (s *UserStore) SaveJourney(ctx context.Context, journey CourseJourney) error {
	if s == nil || s.db == nil {
		return errors.New("opening user store is required")
	}
	if err := validateCourseJourney(journey); err != nil {
		return err
	}
	pathJSON, err := json.Marshal(journey.PathLessonIDs)
	if err != nil {
		return fmt.Errorf("encode opening course journey path: %w", err)
	}
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO opening_course_journeys(
		   course_id, depth, current_lesson_id, current_activity_id,
		   path_lesson_ids_json, last_recommended_lesson_id, active_session_id,
		   created_at, updated_at
		 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(course_id) DO UPDATE SET
		   depth = excluded.depth,
		   current_lesson_id = excluded.current_lesson_id,
		   current_activity_id = excluded.current_activity_id,
		   path_lesson_ids_json = excluded.path_lesson_ids_json,
		   last_recommended_lesson_id = excluded.last_recommended_lesson_id,
		   active_session_id = excluded.active_session_id,
		   created_at = opening_course_journeys.created_at,
		   updated_at = excluded.updated_at`,
		journey.CourseID,
		journey.Depth,
		journey.CurrentLessonID,
		journey.CurrentActivityID,
		string(pathJSON),
		journey.LastRecommendedLessonID,
		journey.ActiveSessionID,
		journey.CreatedAt.UnixMilli(),
		journey.UpdatedAt.UnixMilli(),
	)
	return err
}

func validateCourseJourney(journey CourseJourney) error {
	if err := validateCourseKey(journey.CourseID); err != nil {
		return err
	}
	if _, ok := depthRank(journey.Depth); !ok {
		return fmt.Errorf("invalid opening journey depth %q", journey.Depth)
	}
	if journey.CreatedAt.IsZero() || journey.UpdatedAt.IsZero() {
		return errors.New("opening journey timestamps are required")
	}
	if journey.UpdatedAt.Before(journey.CreatedAt) {
		return errors.New("opening journey update cannot precede creation")
	}
	seen := make(map[string]struct{}, len(journey.PathLessonIDs))
	for _, lessonID := range journey.PathLessonIDs {
		if strings.TrimSpace(lessonID) == "" {
			return errors.New("opening journey path lesson ID is required")
		}
		if _, duplicate := seen[lessonID]; duplicate {
			return fmt.Errorf("opening journey path lesson ID %q is duplicated", lessonID)
		}
		seen[lessonID] = struct{}{}
	}
	return nil
}
