package openings

import (
	"strings"
	"testing"
	"time"
)

func validExplicitStoredSession(now time.Time) StoredSession {
	return StoredSession{
		ID: "session-1", CourseID: "italian-white", GenerationID: "generation-1",
		LessonID: "giuoco-c3", Mode: OpeningModeLesson, Status: OpeningStatusActive,
		Depth: DepthReference,
		State: SessionState{
			Position: PositionState{PositionID: "after-bc5", CurrentFEN: "fen-after-bc5"},
			Attempt: &AttemptState{
				AttemptID: "attempt-1", PromptID: "recall-c3", StartedAt: now,
			},
		},
	}
}

func TestValidateStoredSessionRejectsImpossibleStateCombinations(t *testing.T) {
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		mutate  func(*StoredSession)
		wantErr string
	}{
		{
			name: "lesson with review cursor",
			mutate: func(session *StoredSession) {
				session.State.Review = &ReviewCursor{PromptIDs: []string{"recall-c3"}}
			},
			wantErr: "lesson session cannot carry a review cursor",
		},
		{
			name: "review without cursor",
			mutate: func(session *StoredSession) {
				session.Mode = OpeningModeReview
				session.LessonID = "review"
			},
			wantErr: "review session requires a review cursor",
		},
		{
			name: "restart required with attempt",
			mutate: func(session *StoredSession) {
				session.Status = OpeningStatusRestartRequired
				session.State.Restart = &RestartCheckpoint{ActivityIndex: 1}
			},
			wantErr: "restart-required session cannot carry an attempt",
		},
		{
			name: "negative review index",
			mutate: func(session *StoredSession) {
				session.Mode = OpeningModeReview
				session.LessonID = "review"
				session.State.Review = &ReviewCursor{PromptIDs: []string{"recall-c3"}, Index: -1}
			},
			wantErr: "opening review index cannot be negative",
		},
		{
			name: "negative attempt metric",
			mutate: func(session *StoredSession) {
				session.State.Attempt.IncorrectMoves = -1
			},
			wantErr: "opening attempt metrics cannot be negative",
		},
		{
			name: "negative restart checkpoint",
			mutate: func(session *StoredSession) {
				session.State.Attempt = nil
				session.State.Restart = &RestartCheckpoint{ActivityIndex: -1}
			},
			wantErr: "opening restart activity index cannot be negative",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session := validExplicitStoredSession(now)
			test.mutate(&session)
			err := validateStoredSession(session)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("validateStoredSession() error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestAttemptRecordRequiresActiveAttempt(t *testing.T) {
	if _, err := attemptRecord(nil); err == nil || err.Error() != "active opening prompt requires an attempt" {
		t.Fatalf("attemptRecord(nil) error = %v", err)
	}

	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	record, err := attemptRecord(&AttemptState{
		AttemptID: "attempt-1", PromptID: "recall-c3", StartedAt: now,
		IncorrectMoves: 1, AlternativesTried: 2, HintsUsed: 3, Revealed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.AttemptID != "attempt-1" || record.PromptID != "recall-c3" ||
		record.StartedAt != now || record.IncorrectMoves != 1 ||
		record.AlternativesTried != 2 || record.HintsUsed != 3 || !record.Revealed {
		t.Fatalf("attemptRecord() = %+v", record)
	}
}

func TestValidatePromptCompletionRejectsZeroAttemptRecord(t *testing.T) {
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	err := validatePromptCompletion(PromptCompletion{
		Session: validExplicitStoredSession(now), SemanticFingerprint: "semantic-v1",
		Outcome: ReviewClean,
	}, now.Add(time.Minute))
	if err == nil || err.Error() != "opening attempt ID is required" {
		t.Fatalf("validatePromptCompletion() error = %v", err)
	}
}
