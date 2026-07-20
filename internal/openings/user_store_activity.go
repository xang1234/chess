package openings

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (s *UserStore) CreateLessonSession(
	ctx context.Context,
	seed SessionSeed,
	journey CourseJourney,
	now time.Time,
) (StoredSession, CourseJourney, error) {
	if s == nil || s.db == nil {
		return StoredSession{}, CourseJourney{}, errors.New("opening user store is required")
	}
	if err := validateSessionSeed(seed); err != nil {
		return StoredSession{}, CourseJourney{}, err
	}
	if seed.Mode != OpeningModeLesson {
		return StoredSession{}, CourseJourney{}, errors.New("lesson-session creation requires lesson mode")
	}
	if now.IsZero() {
		return StoredSession{}, CourseJourney{}, errors.New("opening session time is required")
	}
	if seed.CourseID != journey.CourseID || seed.LessonID != journey.CurrentLessonID {
		return StoredSession{}, CourseJourney{}, errors.New("opening lesson session and journey must identify the same course and lesson")
	}
	stateJSON, err := encodeSessionState(seed.State)
	if err != nil {
		return StoredSession{}, CourseJourney{}, err
	}
	session := StoredSession{
		ID: uuid.NewString(), CourseID: seed.CourseID, GenerationID: seed.GenerationID,
		LessonID: seed.LessonID, Mode: seed.Mode, Status: OpeningStatusActive,
		Depth: seed.Depth, ActivityIndex: seed.ActivityIndex, State: seed.State,
	}
	journey.UpdatedAt = now
	if journey.CreatedAt.IsZero() {
		journey.CreatedAt = now
	}
	if err := validateStoredSession(session); err != nil {
		return StoredSession{}, CourseJourney{}, err
	}
	if err := validateCourseJourney(journey); err != nil {
		return StoredSession{}, CourseJourney{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return StoredSession{}, CourseJourney{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE opening_sessions SET status = 'completed', updated_at = ?
		 WHERE status IN ('active','paused','restart_required')`,
		now.UnixMilli(),
	); err != nil {
		return StoredSession{}, CourseJourney{}, fmt.Errorf("retire previous opening session: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO opening_sessions(
		   session_id, course_id, generation_id, lesson_id, mode, status,
		   depth, activity_index, state_json, created_at, updated_at
		 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		session.ID, session.CourseID, session.GenerationID, session.LessonID,
		session.Mode, session.Status, session.Depth, session.ActivityIndex,
		stateJSON, now.UnixMilli(), now.UnixMilli(),
	); err != nil {
		return StoredSession{}, CourseJourney{}, fmt.Errorf("insert opening lesson session: %w", err)
	}
	if err := saveJourneyTx(ctx, tx, journey); err != nil {
		return StoredSession{}, CourseJourney{}, fmt.Errorf("save opening lesson journey: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return StoredSession{}, CourseJourney{}, err
	}
	return session, journey, nil
}

func (s *UserStore) CompleteLessonActivity(
	ctx context.Context,
	completion LessonActivityCompletion,
) error {
	if s == nil || s.db == nil {
		return errors.New("opening user store is required")
	}
	if err := validateLessonActivityCompletion(completion); err != nil {
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

	if completion.Attempt != nil {
		promptCompletion := PromptCompletion{
			Session: completion.Session, Attempt: *completion.Attempt,
			SemanticFingerprint: completion.SemanticFingerprint, Outcome: completion.Outcome,
		}
		if err := recordAttemptAndReviewTx(ctx, tx, promptCompletion, completion.Now); err != nil {
			return err
		}
	}
	if err := recordActivityProgressTx(ctx, tx, ActivityProgressUpdate{
		CourseID: completion.Session.CourseID, LessonID: completion.Session.LessonID,
		CompletedActivityID: completion.ActivityID,
		RequiredActivityIDs: completion.RequiredActivityIDs,
		Now:                 completion.Now,
	}); err != nil {
		return err
	}
	result, err := tx.ExecContext(
		ctx,
		`UPDATE opening_sessions
		 SET status = ?, depth = ?, activity_index = ?, state_json = ?, updated_at = ?
		 WHERE session_id = ? AND course_id = ? AND generation_id = ?
		   AND lesson_id = ? AND mode = ?`,
		completion.Session.Status, completion.Session.Depth, completion.Session.ActivityIndex,
		stateJSON, completion.Now.UnixMilli(), completion.Session.ID,
		completion.Session.CourseID, completion.Session.GenerationID,
		completion.Session.LessonID, completion.Session.Mode,
	)
	if err != nil {
		return fmt.Errorf("update opening session after activity: %w", err)
	}
	if err := requireOneSessionRow(result, completion.Session.ID); err != nil {
		return err
	}
	if err := saveJourneyTx(ctx, tx, completion.Journey); err != nil {
		return fmt.Errorf("save opening journey after activity: %w", err)
	}
	return tx.Commit()
}

func validateLessonActivityCompletion(completion LessonActivityCompletion) error {
	if completion.Session.Mode != OpeningModeLesson {
		return errors.New("activity completion requires a lesson session")
	}
	if err := validateStoredSession(completion.Session); err != nil {
		return err
	}
	if completion.Journey.CourseID != completion.Session.CourseID {
		return errors.New("opening activity session and journey course IDs must match")
	}
	if err := validateCourseJourney(completion.Journey); err != nil {
		return err
	}
	update := ActivityProgressUpdate{
		CourseID: completion.Session.CourseID, LessonID: completion.Session.LessonID,
		CompletedActivityID: completion.ActivityID,
		RequiredActivityIDs: completion.RequiredActivityIDs,
		Now:                 completion.Now,
	}
	if err := validateActivityProgressUpdate(update); err != nil {
		return err
	}
	if completion.Attempt == nil {
		if completion.SemanticFingerprint != "" || completion.Outcome != "" {
			return errors.New("passive opening activity cannot carry review data")
		}
		return nil
	}
	return validatePromptCompletion(PromptCompletion{
		Session: completion.Session, Attempt: *completion.Attempt,
		SemanticFingerprint: completion.SemanticFingerprint, Outcome: completion.Outcome,
	}, completion.Now)
}
