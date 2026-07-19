package openings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *UserStore) CreateSession(
	ctx context.Context,
	seed SessionSeed,
	now time.Time,
) (StoredSession, error) {
	if s == nil || s.db == nil {
		return StoredSession{}, errors.New("opening user store is required")
	}
	if err := validateSessionSeed(seed); err != nil {
		return StoredSession{}, err
	}
	if now.IsZero() {
		return StoredSession{}, errors.New("opening session time is required")
	}
	stateJSON, err := encodeSessionState(seed.State)
	if err != nil {
		return StoredSession{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return StoredSession{}, err
	}
	defer tx.Rollback()
	var existing string
	if err := tx.QueryRowContext(
		ctx,
		`SELECT session_id FROM opening_sessions
		 WHERE status IN ('active','paused','restart_required') LIMIT 1`,
	).Scan(&existing); err == nil {
		return StoredSession{}, ErrResumableSessionExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return StoredSession{}, err
	}
	session := StoredSession{
		ID: uuid.NewString(), CourseID: seed.CourseID, GenerationID: seed.GenerationID,
		LessonID: seed.LessonID, Mode: seed.Mode, Status: OpeningStatusActive,
		Depth: seed.Depth, State: seed.State,
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO opening_sessions(
		   session_id, course_id, generation_id, lesson_id, mode, status,
		   depth, step_index, state_json, created_at, updated_at
		 ) VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?)`,
		session.ID,
		session.CourseID,
		session.GenerationID,
		session.LessonID,
		session.Mode,
		session.Status,
		session.Depth,
		stateJSON,
		now.UnixMilli(),
		now.UnixMilli(),
	); err != nil {
		if strings.Contains(err.Error(), "idx_opening_sessions_single_resumable") {
			return StoredSession{}, ErrResumableSessionExists
		}
		return StoredSession{}, err
	}
	if err := tx.Commit(); err != nil {
		return StoredSession{}, err
	}
	return session, nil
}

func (s *UserStore) LoadSession(
	ctx context.Context,
	sessionID string,
) (StoredSession, error) {
	return scanOpeningSession(s.db.QueryRowContext(
		ctx,
		`SELECT session_id, course_id, generation_id, lesson_id, mode, status,
		        depth, step_index, state_json
		 FROM opening_sessions WHERE session_id = ?`,
		sessionID,
	))
}

func (s *UserStore) ResumableSession(ctx context.Context) (*StoredSession, error) {
	session, err := scanOpeningSession(s.db.QueryRowContext(
		ctx,
		`SELECT session_id, course_id, generation_id, lesson_id, mode, status,
		        depth, step_index, state_json
		 FROM opening_sessions
		 WHERE status IN ('active','paused','restart_required')
		 ORDER BY updated_at DESC LIMIT 1`,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *UserStore) SaveSession(
	ctx context.Context,
	session StoredSession,
	now time.Time,
) error {
	if err := validateStoredSession(session); err != nil {
		return err
	}
	stateJSON, err := encodeSessionState(session.State)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE opening_sessions
		 SET status = ?, depth = ?, step_index = ?, state_json = ?, updated_at = ?
		 WHERE session_id = ? AND course_id = ? AND generation_id = ?
		   AND lesson_id = ? AND mode = ?`,
		session.Status,
		session.Depth,
		session.StepIndex,
		stateJSON,
		now.UnixMilli(),
		session.ID,
		session.CourseID,
		session.GenerationID,
		session.LessonID,
		session.Mode,
	)
	if err != nil {
		return err
	}
	return requireOneSessionRow(result, session.ID)
}

func (s *UserStore) SetSessionStatus(
	ctx context.Context,
	sessionID string,
	status OpeningSessionStatus,
	now time.Time,
) error {
	if strings.TrimSpace(sessionID) == "" {
		return errors.New("opening session ID is required")
	}
	if !validOpeningStatus(status) {
		return fmt.Errorf("invalid opening session status %q", status)
	}
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE opening_sessions SET status = ?, updated_at = ? WHERE session_id = ?`,
		status,
		now.UnixMilli(),
		sessionID,
	)
	if err != nil {
		return err
	}
	return requireOneSessionRow(result, sessionID)
}

type rowScanner interface {
	Scan(...any) error
}

func scanOpeningSession(row rowScanner) (StoredSession, error) {
	var session StoredSession
	var stateJSON string
	if err := row.Scan(
		&session.ID,
		&session.CourseID,
		&session.GenerationID,
		&session.LessonID,
		&session.Mode,
		&session.Status,
		&session.Depth,
		&session.StepIndex,
		&stateJSON,
	); err != nil {
		return StoredSession{}, err
	}
	if err := json.Unmarshal([]byte(stateJSON), &session.State); err != nil {
		return StoredSession{}, fmt.Errorf("decode opening session state: %w", err)
	}
	return session, nil
}

func encodeSessionState(state SessionState) (string, error) {
	encoded, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("encode opening session state: %w", err)
	}
	return string(encoded), nil
}

func validateSessionSeed(seed SessionSeed) error {
	if err := validateCourseKey(seed.CourseID); err != nil {
		return err
	}
	if strings.TrimSpace(seed.GenerationID) == "" {
		return errors.New("course generation ID is required")
	}
	if strings.TrimSpace(seed.LessonID) == "" {
		return errors.New("opening lesson ID is required")
	}
	if !validOpeningMode(seed.Mode) {
		return fmt.Errorf("invalid opening session mode %q", seed.Mode)
	}
	if _, ok := depthRank(seed.Depth); !ok {
		return fmt.Errorf("invalid opening depth %q", seed.Depth)
	}
	return nil
}

func validateStoredSession(session StoredSession) error {
	if strings.TrimSpace(session.ID) == "" {
		return errors.New("opening session ID is required")
	}
	if err := validateSessionSeed(SessionSeed{
		CourseID: session.CourseID, GenerationID: session.GenerationID,
		LessonID: session.LessonID, Mode: session.Mode, Depth: session.Depth,
	}); err != nil {
		return err
	}
	if !validOpeningStatus(session.Status) {
		return fmt.Errorf("invalid opening session status %q", session.Status)
	}
	if session.StepIndex < 0 {
		return errors.New("opening session step index cannot be negative")
	}
	return nil
}

func requireOneSessionRow(result sql.Result, sessionID string) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("opening session %q was not found", sessionID)
	}
	return nil
}
