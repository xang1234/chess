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

var ErrResumableSessionExists = errors.New("a resumable opening session already exists")

type OpeningSessionMode string

const (
	OpeningModeLesson OpeningSessionMode = "lesson"
	OpeningModeReview OpeningSessionMode = "review"
)

type OpeningSessionStatus string

const (
	OpeningStatusActive          OpeningSessionStatus = "active"
	OpeningStatusPaused          OpeningSessionStatus = "paused"
	OpeningStatusCompleted       OpeningSessionStatus = "completed"
	OpeningStatusRestartRequired OpeningSessionStatus = "restart_required"
)

type PositionState struct {
	PositionID    string   `json:"positionId"`
	CurrentFEN    string   `json:"currentFen"`
	PlayedMoveIDs []string `json:"playedMoveIds"`
}

type AttemptState struct {
	AttemptID         string    `json:"attemptId"`
	PromptID          string    `json:"promptId"`
	StartedAt         time.Time `json:"startedAt"`
	HintLevel         int       `json:"hintLevel"`
	IncorrectMoves    int       `json:"incorrectMoves"`
	AlternativesTried int       `json:"alternativesTried"`
	HintsUsed         int       `json:"hintsUsed"`
	Revealed          bool      `json:"revealed"`
}

type ReviewCursor struct {
	PromptIDs []string `json:"promptIds"`
	Index     int      `json:"index"`
}

type SessionSummary struct {
	CompletedPrompts   int `json:"completedPrompts"`
	PositionsRecalled  int `json:"positionsRecalled"`
	BranchesRecognized int `json:"branchesRecognized"`
	Retried            int `json:"retried"`
	UsedHint           int `json:"usedHint"`
	Revealed           int `json:"revealed"`
}

type RestartCheckpoint struct {
	ActivityIndex int `json:"activityIndex"`
}

type SessionState struct {
	Position PositionState      `json:"position"`
	Attempt  *AttemptState      `json:"attempt,omitempty"`
	Review   *ReviewCursor      `json:"review,omitempty"`
	Summary  SessionSummary     `json:"summary"`
	Restart  *RestartCheckpoint `json:"restart,omitempty"`
}

type StoredSession struct {
	ID            string
	CourseID      string
	GenerationID  string
	LessonID      string
	Mode          OpeningSessionMode
	Status        OpeningSessionStatus
	Depth         Depth
	ActivityIndex int
	State         SessionState
}

type SessionSeed struct {
	CourseID      string
	GenerationID  string
	LessonID      string
	Mode          OpeningSessionMode
	Depth         Depth
	ActivityIndex int
	State         SessionState
}

type LessonProgress struct {
	CourseID             string
	LessonID             string
	CompletedActivityIDs []string
	CompletedActivities  int
	TotalActivities      int
	Completed            bool
}

type CourseJourney struct {
	CourseID        string
	CurrentLessonID string
	PathLessonIDs   []string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ActivityProgressUpdate struct {
	CourseID            string
	LessonID            string
	CompletedActivityID string
	RequiredActivityIDs []string
	Now                 time.Time
}

type LessonActivityCompletion struct {
	Session             StoredSession
	Journey             CourseJourney
	ActivityID          string
	RequiredActivityIDs []string
	Attempt             *AttemptRecord
	SemanticFingerprint string
	Outcome             ReviewOutcome
	Now                 time.Time
}

type ReviewOutcome = spacedreview.Outcome

const (
	ReviewClean    = spacedreview.Clean
	ReviewMissed   = spacedreview.Missed
	ReviewHinted   = spacedreview.Hinted
	ReviewRevealed = spacedreview.Revealed
)

type ReviewState struct {
	CourseID            string
	PromptID            string
	SemanticFingerprint string
	DueAt               time.Time
	IntervalIndex       int
	SuccessfulReviews   int
	LastOutcome         ReviewOutcome
	Status              string
}

type PromptCompletion struct {
	Session              StoredSession
	Attempt              AttemptRecord
	SemanticFingerprint  string
	Outcome              ReviewOutcome
	CompletedActivityIDs []string
}

type AttemptRecord struct {
	AttemptID         string
	PromptID          string
	StartedAt         time.Time
	IncorrectMoves    int
	AlternativesTried int
	HintsUsed         int
	Revealed          bool
}

func attemptRecord(attempt *AttemptState) (AttemptRecord, error) {
	if attempt == nil {
		return AttemptRecord{}, errors.New("active opening prompt requires an attempt")
	}
	return AttemptRecord{
		AttemptID:         attempt.AttemptID,
		PromptID:          attempt.PromptID,
		StartedAt:         attempt.StartedAt,
		IncorrectMoves:    attempt.IncorrectMoves,
		AlternativesTried: attempt.AlternativesTried,
		HintsUsed:         attempt.HintsUsed,
		Revealed:          attempt.Revealed,
	}, nil
}

type CourseRevision struct {
	CourseID           string
	PromptFingerprints map[string]string
	SessionRebase      *SessionRebase
	Journey            *CourseJourney
	Now                time.Time
}

type SessionRebase struct {
	PreviousGenerationID string
	Session              StoredSession
}

type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) Depth(
	ctx context.Context,
	courseID string,
	fallback Depth,
) (Depth, error) {
	if err := validateCourseKey(courseID); err != nil {
		return "", err
	}
	if _, ok := depthRank(fallback); !ok {
		return "", fmt.Errorf("invalid fallback opening depth %q", fallback)
	}
	var depth Depth
	err := s.db.QueryRowContext(
		ctx,
		`SELECT depth FROM opening_preferences WHERE course_id = ?`,
		courseID,
	).Scan(&depth)
	if errors.Is(err, sql.ErrNoRows) {
		return fallback, nil
	}
	return depth, err
}

func (s *UserStore) SetDepth(
	ctx context.Context,
	courseID string,
	depth Depth,
	now time.Time,
) error {
	if err := validateCourseKey(courseID); err != nil {
		return err
	}
	if _, ok := depthRank(depth); !ok {
		return fmt.Errorf("invalid opening depth %q", depth)
	}
	if now.IsZero() {
		return errors.New("opening preference time is required")
	}
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO opening_preferences(course_id, depth, updated_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(course_id) DO UPDATE SET
		   depth = excluded.depth,
		   updated_at = excluded.updated_at`,
		courseID,
		depth,
		now.UnixMilli(),
	)
	return err
}

func validateCourseKey(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("course ID is required")
	}
	return nil
}

func validOpeningMode(mode OpeningSessionMode) bool {
	return mode == OpeningModeLesson || mode == OpeningModeReview
}

func resumableOpeningStatus(status OpeningSessionStatus) bool {
	switch status {
	case OpeningStatusActive, OpeningStatusPaused, OpeningStatusRestartRequired:
		return true
	default:
		return false
	}
}

func validOpeningStatus(status OpeningSessionStatus) bool {
	return resumableOpeningStatus(status) || status == OpeningStatusCompleted
}
