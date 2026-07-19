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

type SessionState struct {
	PositionID        string    `json:"positionId"`
	CurrentFEN        string    `json:"currentFen"`
	PlayedMoveIDs     []string  `json:"playedMoveIds"`
	ReviewPromptIDs   []string  `json:"reviewPromptIds,omitempty"`
	ReviewIndex       int       `json:"reviewIndex,omitempty"`
	HintLevel         int       `json:"hintLevel"`
	IncorrectMoves    int       `json:"incorrectMoves"`
	AlternativesTried int       `json:"alternativesTried"`
	HintsUsed         int       `json:"hintsUsed"`
	Revealed          bool      `json:"revealed"`
	AttemptID         string    `json:"attemptId"`
	PromptID          string    `json:"promptId,omitempty"`
	StartedAt         time.Time `json:"startedAt"`
}

type StoredSession struct {
	ID           string
	CourseID     string
	GenerationID string
	LessonID     string
	Mode         OpeningSessionMode
	Status       OpeningSessionStatus
	Depth        Depth
	StepIndex    int
	State        SessionState
}

type SessionSeed struct {
	CourseID     string
	GenerationID string
	LessonID     string
	Mode         OpeningSessionMode
	Depth        Depth
	State        SessionState
}

type LessonProgress struct {
	CourseID         string
	LessonID         string
	CompletedStepIDs []string
	CompletedSteps   int
	TotalSteps       int
	Completed        bool
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
	Session             StoredSession
	SemanticFingerprint string
	Outcome             ReviewOutcome
	CompletedStepIDs    []string
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
