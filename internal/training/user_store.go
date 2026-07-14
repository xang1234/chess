package training

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type UserStore struct {
	db *sql.DB
}

type itemState struct {
	CurrentFEN     string        `json:"currentFen"`
	Path           []int         `json:"path"`
	HintLevel      int           `json:"hintLevel"`
	IncorrectMoves int           `json:"incorrectMoves"`
	HintsUsed      int           `json:"hintsUsed"`
	Revealed       bool          `json:"revealed"`
	Completed      bool          `json:"completed"`
	Unavailable    bool          `json:"unavailable"`
	Kind           ScheduledKind `json:"kind"`
	UpdatesRating  bool          `json:"updatesRating"`
	AttemptID      string        `json:"attemptId"`
	StartedAt      time.Time     `json:"startedAt"`
}

type storedItem struct {
	Ordinal     int
	Fingerprint string
	SourceID    string
	State       itemState
}

type storedSession struct {
	ID           string
	Mode         string
	Status       string
	CurrentIndex int
	Items        []storedItem
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) UpdateProfile(ctx context.Context, profile Profile) error {
	if profile.SessionSize != 5 && profile.SessionSize != 10 && profile.SessionSize != 15 {
		return fmt.Errorf("unsupported session size %d", profile.SessionSize)
	}
	now := time.Now().UnixMilli()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var previous float64
	previousErr := tx.QueryRowContext(ctx, `SELECT learner_rating FROM profile WHERE id = 1`).Scan(&previous)
	if previousErr != nil && !errors.Is(previousErr, sql.ErrNoRows) {
		return previousErr
	}
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO profile(id, learner_rating, session_size, created_at, updated_at)
         VALUES (1, ?, ?, ?, ?)
         ON CONFLICT(id) DO UPDATE SET
           learner_rating=excluded.learner_rating,
           session_size=excluded.session_size,
           updated_at=excluded.updated_at`,
		profile.LearnerRating,
		profile.SessionSize,
		now,
		now,
	)
	if err != nil {
		return err
	}
	if errors.Is(previousErr, sql.ErrNoRows) || previous != profile.LearnerRating {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO rating_history(rating, recorded_at) VALUES (?, ?)`,
			profile.LearnerRating, now,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *UserStore) Profile(ctx context.Context) (Profile, error) {
	var profile Profile
	err := s.db.QueryRowContext(
		ctx,
		`SELECT learner_rating, session_size FROM profile WHERE id = 1`,
	).Scan(&profile.LearnerRating, &profile.SessionSize)
	return profile, err
}

func (s *UserStore) DueReviews(ctx context.Context, now time.Time, limit int) ([]ReviewState, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT fingerprint, due_at, interval_index, successful_reviews, last_outcome
         FROM review_state WHERE due_at <= ? ORDER BY due_at LIMIT ?`,
		now.Unix(),
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
			&state.Fingerprint,
			&dueAt,
			&state.IntervalIndex,
			&state.SuccessfulReviews,
			&state.LastOutcome,
		); err != nil {
			return nil, err
		}
		state.DueAt = time.Unix(dueAt, 0)
		states = append(states, state)
	}
	return states, rows.Err()
}

func (s *UserStore) RecentFingerprints(ctx context.Context, limit int) ([]string, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT fingerprint FROM attempts
         GROUP BY fingerprint ORDER BY MAX(started_at) DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	fingerprints := make([]string, 0, limit)
	for rows.Next() {
		var fingerprint string
		if err := rows.Scan(&fingerprint); err != nil {
			return nil, err
		}
		fingerprints = append(fingerprints, fingerprint)
	}
	return fingerprints, rows.Err()
}

func (s *UserStore) CreateSession(
	ctx context.Context,
	mode string,
	items []ScheduledPuzzle,
	now time.Time,
) (storedSession, error) {
	if len(items) == 0 {
		return storedSession{}, errors.New("cannot create an empty session")
	}
	session := storedSession{
		ID:     uuid.NewString(),
		Mode:   mode,
		Status: "active",
		Items:  make([]storedItem, len(items)),
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return storedSession{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO sessions(session_id, mode, status, created_at, updated_at, current_index)
         VALUES (?, ?, 'active', ?, ?, 0)`,
		session.ID,
		mode,
		now.UnixMilli(),
		now.UnixMilli(),
	); err != nil {
		return storedSession{}, err
	}
	for index, scheduled := range items {
		startedAt := time.Time{}
		if index == 0 {
			startedAt = now
		}
		state := itemState{
			CurrentFEN:    scheduled.Puzzle.DisplayedFEN,
			Path:          []int{},
			Kind:          scheduled.Kind,
			UpdatesRating: scheduled.UpdatesRating,
			AttemptID:     uuid.NewString(),
			StartedAt:     startedAt,
		}
		encoded, err := json.Marshal(state)
		if err != nil {
			return storedSession{}, err
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO session_items(session_id, ordinal, fingerprint, source_id, state_json)
             VALUES (?, ?, ?, ?, ?)`,
			session.ID,
			index,
			scheduled.Puzzle.Fingerprint,
			scheduled.SourceID,
			string(encoded),
		); err != nil {
			return storedSession{}, err
		}
		session.Items[index] = storedItem{
			Ordinal:     index,
			Fingerprint: scheduled.Puzzle.Fingerprint,
			SourceID:    scheduled.SourceID,
			State:       state,
		}
	}
	if err := insertAttempt(ctx, tx, session.ID, session.Items[0]); err != nil {
		return storedSession{}, err
	}
	if err := tx.Commit(); err != nil {
		return storedSession{}, err
	}
	return session, nil
}

func insertAttempt(ctx context.Context, tx *sql.Tx, sessionID string, item storedItem) error {
	_, err := tx.ExecContext(
		ctx,
		`INSERT OR IGNORE INTO attempts(
           attempt_id, session_id, fingerprint, source_id, started_at,
           incorrect_moves, hints_used, solution_revealed, first_try, duration_ms
         ) VALUES (?, ?, ?, ?, ?, 0, 0, 0, 0, 0)`,
		item.State.AttemptID,
		sessionID,
		item.Fingerprint,
		item.SourceID,
		item.State.StartedAt.UnixMilli(),
	)
	return err
}

func (s *UserStore) LoadSession(ctx context.Context, sessionID string) (storedSession, error) {
	var session storedSession
	err := s.db.QueryRowContext(
		ctx,
		`SELECT session_id, mode, status, current_index FROM sessions WHERE session_id = ?`,
		sessionID,
	).Scan(&session.ID, &session.Mode, &session.Status, &session.CurrentIndex)
	if err != nil {
		return storedSession{}, err
	}
	if err := s.loadItems(ctx, &session); err != nil {
		return storedSession{}, err
	}
	return session, nil
}

func (s *UserStore) ResumableSession(ctx context.Context) (*storedSession, error) {
	var sessionID string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT session_id FROM sessions
         WHERE status IN ('active', 'paused') ORDER BY updated_at DESC LIMIT 1`,
	).Scan(&sessionID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	session, err := s.LoadSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *UserStore) loadItems(ctx context.Context, session *storedSession) error {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT ordinal, fingerprint, source_id, state_json
         FROM session_items WHERE session_id = ? ORDER BY ordinal`,
		session.ID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item storedItem
		var encoded string
		if err := rows.Scan(&item.Ordinal, &item.Fingerprint, &item.SourceID, &encoded); err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(encoded), &item.State); err != nil {
			return err
		}
		session.Items = append(session.Items, item)
	}
	return rows.Err()
}

func (s *UserStore) SaveItemState(
	ctx context.Context,
	sessionID string,
	ordinal int,
	state itemState,
	now time.Time,
) error {
	encoded, err := json.Marshal(state)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE session_items SET state_json = ? WHERE session_id = ? AND ordinal = ?`,
		string(encoded),
		sessionID,
		ordinal,
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE attempts SET incorrect_moves = ?, hints_used = ?, solution_revealed = ?
         WHERE attempt_id = ?`,
		state.IncorrectMoves,
		state.HintsUsed,
		boolInteger(state.Revealed),
		state.AttemptID,
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE sessions SET updated_at = ? WHERE session_id = ?`,
		now.UnixMilli(),
		sessionID,
	); err != nil {
		return err
	}
	return tx.Commit()
}

func boolInteger(value bool) int {
	if value {
		return 1
	}
	return 0
}

type completionEffects struct {
	Review    *ReviewState
	NewRating *float64
}

func (s *UserStore) HasCompletedAttemptBefore(
	ctx context.Context,
	fingerprint string,
	attemptID string,
) (bool, error) {
	var count int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM attempts
         WHERE fingerprint = ? AND attempt_id <> ? AND completed_at IS NOT NULL`,
		fingerprint,
		attemptID,
	).Scan(&count)
	return count > 0, err
}

func (s *UserStore) Review(ctx context.Context, fingerprint string) (ReviewState, bool, error) {
	var state ReviewState
	var dueAt int64
	err := s.db.QueryRowContext(
		ctx,
		`SELECT fingerprint, due_at, interval_index, successful_reviews, last_outcome
         FROM review_state WHERE fingerprint = ?`,
		fingerprint,
	).Scan(
		&state.Fingerprint,
		&dueAt,
		&state.IntervalIndex,
		&state.SuccessfulReviews,
		&state.LastOutcome,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ReviewState{Fingerprint: fingerprint}, false, nil
	}
	if err != nil {
		return ReviewState{}, false, err
	}
	state.DueAt = time.Unix(dueAt, 0)
	return state, true, nil
}

func (s *UserStore) CompleteItem(
	ctx context.Context,
	session storedSession,
	state itemState,
	now time.Time,
	effects completionEffects,
) (storedSession, error) {
	if session.CurrentIndex >= len(session.Items) {
		return storedSession{}, errors.New("session is already complete")
	}
	current := session.Items[session.CurrentIndex]
	current.State = state
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return storedSession{}, err
	}
	defer tx.Rollback()
	if err := writeItemState(ctx, tx, session.ID, current.Ordinal, state); err != nil {
		return storedSession{}, err
	}
	duration := now.Sub(state.StartedAt).Milliseconds()
	if duration < 0 {
		duration = 0
	}
	firstTry := state.IncorrectMoves == 0 && state.HintsUsed == 0 && !state.Revealed
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE attempts SET
           completed_at = ?, incorrect_moves = ?, hints_used = ?, solution_revealed = ?,
           first_try = ?, duration_ms = ?
         WHERE attempt_id = ?`,
		now.UnixMilli(),
		state.IncorrectMoves,
		state.HintsUsed,
		boolInteger(state.Revealed),
		boolInteger(firstTry),
		duration,
		state.AttemptID,
	); err != nil {
		return storedSession{}, err
	}
	if effects.Review != nil {
		review := effects.Review
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO review_state(
               fingerprint, due_at, interval_index, successful_reviews, last_outcome
             ) VALUES (?, ?, ?, ?, ?)
             ON CONFLICT(fingerprint) DO UPDATE SET
               due_at=excluded.due_at,
               interval_index=excluded.interval_index,
               successful_reviews=excluded.successful_reviews,
               last_outcome=excluded.last_outcome`,
			review.Fingerprint,
			review.DueAt.Unix(),
			review.IntervalIndex,
			review.SuccessfulReviews,
			review.LastOutcome,
		); err != nil {
			return storedSession{}, err
		}
	}
	if effects.NewRating != nil {
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE profile SET learner_rating = ?, updated_at = ? WHERE id = 1`,
			*effects.NewRating,
			now.UnixMilli(),
		); err != nil {
			return storedSession{}, err
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO rating_history(rating, recorded_at) VALUES (?, ?)`,
			*effects.NewRating,
			now.UnixMilli(),
		); err != nil {
			return storedSession{}, err
		}
	}

	session.Items[session.CurrentIndex] = current
	session.CurrentIndex++
	status := "active"
	if session.CurrentIndex >= len(session.Items) {
		status = "completed"
	} else {
		next := &session.Items[session.CurrentIndex]
		next.State.StartedAt = now
		if err := writeItemState(ctx, tx, session.ID, next.Ordinal, next.State); err != nil {
			return storedSession{}, err
		}
		if err := insertAttempt(ctx, tx, session.ID, *next); err != nil {
			return storedSession{}, err
		}
	}
	session.Status = status
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE sessions SET status = ?, current_index = ?, updated_at = ? WHERE session_id = ?`,
		status,
		session.CurrentIndex,
		now.UnixMilli(),
		session.ID,
	); err != nil {
		return storedSession{}, err
	}
	if err := tx.Commit(); err != nil {
		return storedSession{}, err
	}
	return session, nil
}

func (s *UserStore) SkipUnavailable(
	ctx context.Context,
	session storedSession,
	now time.Time,
) (storedSession, error) {
	if session.CurrentIndex >= len(session.Items) {
		return session, nil
	}
	current := &session.Items[session.CurrentIndex]
	current.State.Unavailable = true
	current.State.Completed = true
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return storedSession{}, err
	}
	defer tx.Rollback()
	if err := writeItemState(ctx, tx, session.ID, current.Ordinal, current.State); err != nil {
		return storedSession{}, err
	}
	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM attempts WHERE attempt_id = ? AND completed_at IS NULL`,
		current.State.AttemptID,
	); err != nil {
		return storedSession{}, err
	}
	session.CurrentIndex++
	status := "active"
	if session.CurrentIndex >= len(session.Items) {
		status = "completed"
	} else {
		next := &session.Items[session.CurrentIndex]
		next.State.StartedAt = now
		if err := writeItemState(ctx, tx, session.ID, next.Ordinal, next.State); err != nil {
			return storedSession{}, err
		}
		if err := insertAttempt(ctx, tx, session.ID, *next); err != nil {
			return storedSession{}, err
		}
	}
	session.Status = status
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE sessions SET status = ?, current_index = ?, updated_at = ? WHERE session_id = ?`,
		status,
		session.CurrentIndex,
		now.UnixMilli(),
		session.ID,
	); err != nil {
		return storedSession{}, err
	}
	if err := tx.Commit(); err != nil {
		return storedSession{}, err
	}
	return session, nil
}

func writeItemState(
	ctx context.Context,
	tx *sql.Tx,
	sessionID string,
	ordinal int,
	state itemState,
) error {
	encoded, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(
		ctx,
		`UPDATE session_items SET state_json = ? WHERE session_id = ? AND ordinal = ?`,
		string(encoded),
		sessionID,
		ordinal,
	)
	return err
}

func (s *UserStore) SetSessionStatus(ctx context.Context, sessionID, status string, now time.Time) error {
	if status != "active" && status != "paused" {
		return fmt.Errorf("unsupported session status %q", status)
	}
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE sessions SET status = ?, updated_at = ? WHERE session_id = ?`,
		status,
		now.UnixMilli(),
		sessionID,
	)
	return err
}
