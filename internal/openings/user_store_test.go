package openings

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"chess-trainer/internal/storage"
)

func openOpeningUserTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "user.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Migrate(db, "user"); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func openingSessionSeed(now time.Time) SessionSeed {
	return SessionSeed{
		CourseID: "italian-white", GenerationID: "generation-1", LessonID: "giuoco-c3",
		Mode: OpeningModeLesson, Depth: DepthReference,
		State: SessionState{
			PositionID: "after-bc5", CurrentFEN: "fen-after-bc5",
			PlayedMoveIDs: []string{"white-e4", "black-e5"}, HintLevel: 1,
			IncorrectMoves: 0, AlternativesTried: 1, HintsUsed: 0,
			AttemptID: "attempt-1", PromptID: "recall-c3", StartedAt: now,
		},
	}
}

func TestUserStoreDepthAndSessionRoundTrip(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	store := NewUserStore(openOpeningUserTestDB(t))

	depth, err := store.Depth(ctx, "italian-white", DepthStandard)
	if err != nil || depth != DepthStandard {
		t.Fatalf("default depth = %q err=%v", depth, err)
	}
	if err := store.SetDepth(ctx, "italian-white", DepthQuick, now); err != nil {
		t.Fatal(err)
	}
	depth, err = store.Depth(ctx, "italian-white", DepthReference)
	if err != nil || depth != DepthQuick {
		t.Fatalf("stored depth = %q err=%v", depth, err)
	}

	session, err := store.CreateSession(ctx, openingSessionSeed(now), now)
	if err != nil {
		t.Fatal(err)
	}
	session.StepIndex = 3
	session.State.CurrentFEN = "fen-after-c3"
	session.State.PlayedMoveIDs = append(session.State.PlayedMoveIDs, "white-c3")
	session.State.ReviewPromptIDs = []string{"recall-c3", "recall-d3"}
	session.State.ReviewIndex = 1
	if err := store.SaveSession(ctx, session, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.LoadSession(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, session) {
		t.Fatalf("loaded session = %#v, want %#v", loaded, session)
	}
	resumable, err := store.ResumableSession(ctx)
	if err != nil || resumable == nil || resumable.ID != session.ID {
		t.Fatalf("resumable = %#v err=%v", resumable, err)
	}
	for _, status := range []OpeningSessionStatus{
		OpeningStatusActive, OpeningStatusPaused, OpeningStatusRestartRequired,
	} {
		if err := store.SetSessionStatus(ctx, session.ID, status, now.Add(2*time.Minute)); err != nil {
			t.Fatal(err)
		}
		if _, err := store.CreateSession(ctx, openingSessionSeed(now), now); !errors.Is(err, ErrResumableSessionExists) {
			t.Fatalf("second session with %q error = %v, want resumable conflict", status, err)
		}
	}
	if err := store.SetSessionStatus(ctx, session.ID, OpeningStatusCompleted, now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	next, err := store.CreateSession(ctx, openingSessionSeed(now), now.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("completed session blocked the next session: %v", err)
	}
	if err := store.SetSessionStatus(ctx, next.ID, OpeningStatusCompleted, now.Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	reviewSeed := openingSessionSeed(now)
	reviewSeed.Mode = OpeningModeReview
	reviewSeed.LessonID = "review"
	reviewSeed.State.ReviewPromptIDs = []string{"recall-c3", "recall-d3"}
	reviewSeed.State.ReviewIndex = 1
	reviewSession, err := store.CreateSession(ctx, reviewSeed, now.Add(6*time.Minute))
	if err != nil || reviewSession.LessonID != "review" || reviewSession.Mode != OpeningModeReview {
		t.Fatalf("review session = %+v err=%v", reviewSession, err)
	}
}

func TestOpeningLessonProgressProjectsStableStepIDs(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	store := NewUserStore(openOpeningUserTestDB(t))
	completed := []string{"explain", "watch", "try", "branch", "recall"}
	if err := store.CompleteLesson(ctx, "italian-white", "giuoco-c3", completed, now); err != nil {
		t.Fatal(err)
	}

	progress, err := store.LessonProgress(ctx, "italian-white", "giuoco-c3", completed)
	if err != nil || !progress.Completed || progress.CompletedSteps != 5 || progress.TotalSteps != 5 {
		t.Fatalf("original progress = %+v err=%v", progress, err)
	}
	updated := []string{"explain", "watch", "inserted", "try", "branch", "recall"}
	progress, err = store.LessonProgress(ctx, "italian-white", "giuoco-c3", updated)
	if err != nil || progress.Completed || progress.CompletedSteps != 5 || progress.TotalSteps != 6 {
		t.Fatalf("updated progress = %+v err=%v", progress, err)
	}
	shortened := []string{"explain", "watch", "try", "branch"}
	progress, err = store.LessonProgress(ctx, "italian-white", "giuoco-c3", shortened)
	if err != nil || progress.Completed || progress.CompletedSteps != 4 || progress.TotalSteps != 4 {
		t.Fatalf("shortened progress = %+v err=%v", progress, err)
	}
}

func TestOpeningCompletePromptUpdatesAttemptReviewProgressAndSessionAtomically(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	db := openOpeningUserTestDB(t)
	store := NewUserStore(db)
	session, err := store.CreateSession(ctx, openingSessionSeed(now), now)
	if err != nil {
		t.Fatal(err)
	}
	session.StepIndex = 5
	session.Status = OpeningStatusCompleted
	completedSteps := []string{"explain", "watch", "try", "branch", "recall"}
	if err := store.CompletePrompt(ctx, PromptCompletion{
		Session: session, SemanticFingerprint: "semantic-v1", Outcome: ReviewClean,
		CompletedStepIDs: completedSteps,
	}, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}

	var incorrect, alternatives, hints, revealed int
	if err := db.QueryRow(
		`SELECT incorrect_moves, alternatives_tried, hints_used, revealed
		 FROM opening_attempts WHERE attempt_id = 'attempt-1'`,
	).Scan(&incorrect, &alternatives, &hints, &revealed); err != nil {
		t.Fatal(err)
	}
	if incorrect != 0 || alternatives != 1 || hints != 0 || revealed != 0 {
		t.Fatalf("attempt metrics = incorrect %d alternatives %d hints %d revealed %d", incorrect, alternatives, hints, revealed)
	}
	var promptFingerprint string
	var promptOutcome ReviewOutcome
	if err := db.QueryRow(
		`SELECT semantic_fingerprint, last_outcome FROM opening_prompt_progress
		 WHERE course_id = ? AND prompt_id = ?`,
		session.CourseID,
		session.State.PromptID,
	).Scan(&promptFingerprint, &promptOutcome); err != nil {
		t.Fatal(err)
	}
	if promptFingerprint != "semantic-v1" || promptOutcome != ReviewClean {
		t.Fatalf("prompt progress = fingerprint %q outcome %q", promptFingerprint, promptOutcome)
	}
	review, err := store.Review(ctx, session.CourseID, session.State.PromptID)
	if err != nil {
		t.Fatal(err)
	}
	if review.IntervalIndex != 0 || review.SuccessfulReviews != 1 ||
		review.DueAt.Sub(now.Add(2*time.Minute)) != 24*time.Hour {
		t.Fatalf("first review = %+v", review)
	}
	progress, err := store.LessonProgress(ctx, session.CourseID, session.LessonID, completedSteps)
	if err != nil || !progress.Completed {
		t.Fatalf("lesson progress = %+v err=%v", progress, err)
	}
	loaded, err := store.LoadSession(ctx, session.ID)
	if err != nil || loaded.Status != OpeningStatusCompleted || loaded.StepIndex != 5 {
		t.Fatalf("completed session = %+v err=%v", loaded, err)
	}

	for index, days := range []int{3, 7, 21, 60} {
		nextNow := review.DueAt
		session.State.AttemptID = "attempt-clean-" + string(rune('a'+index))
		session.State.StartedAt = nextNow.Add(-time.Minute)
		if err := store.CompletePrompt(ctx, PromptCompletion{
			Session: session, SemanticFingerprint: "semantic-v1", Outcome: ReviewClean,
		}, nextNow); err != nil {
			t.Fatal(err)
		}
		review, err = store.Review(ctx, session.CourseID, session.State.PromptID)
		if err != nil || review.DueAt.Sub(nextNow) != time.Duration(days)*24*time.Hour {
			t.Fatalf("clean review %d = %+v err=%v", index, review, err)
		}
	}
	session.State.AttemptID = "attempt-hinted"
	session.State.StartedAt = review.DueAt.Add(-time.Minute)
	session.State.HintsUsed = 1
	if err := store.CompletePrompt(ctx, PromptCompletion{
		Session: session, SemanticFingerprint: "semantic-v1", Outcome: ReviewHinted,
	}, review.DueAt); err != nil {
		t.Fatal(err)
	}
	reset, err := store.Review(ctx, session.CourseID, session.State.PromptID)
	if err != nil || reset.IntervalIndex != 0 || reset.SuccessfulReviews != 0 ||
		reset.DueAt.Sub(review.DueAt) != 24*time.Hour {
		t.Fatalf("hinted reset = %+v err=%v", reset, err)
	}
	due, err := store.DueReviews(ctx, session.CourseID, reset.DueAt, 10)
	if err != nil || len(due) != 1 || due[0].PromptID != session.State.PromptID {
		t.Fatalf("due reviews = %+v err=%v", due, err)
	}
}

func TestOpeningCompletePromptRollsBackWhenSessionIsMissing(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	db := openOpeningUserTestDB(t)
	store := NewUserStore(db)
	seed := openingSessionSeed(now)
	missing := StoredSession{
		ID: "missing-session", CourseID: seed.CourseID, GenerationID: seed.GenerationID,
		LessonID: seed.LessonID, Mode: seed.Mode, Status: OpeningStatusActive,
		Depth: seed.Depth, State: seed.State,
	}
	err := store.CompletePrompt(ctx, PromptCompletion{
		Session: missing, SemanticFingerprint: "semantic-v1", Outcome: ReviewClean,
	}, now)
	if err == nil {
		t.Fatal("CompletePrompt() unexpectedly succeeded")
	}
	for _, table := range []string{
		"opening_attempts", "opening_prompt_progress", "opening_review_state",
	} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s retained %d rows after rollback", table, count)
		}
	}
}

func TestOpeningProtectedGenerationIDsTracksOnlyResumableSessions(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	store := NewUserStore(openOpeningUserTestDB(t))
	session, err := store.CreateSession(ctx, openingSessionSeed(now), now)
	if err != nil {
		t.Fatal(err)
	}
	protected, err := store.ProtectedGenerationIDs(ctx)
	if err != nil || !reflect.DeepEqual(protected, map[string]struct{}{session.GenerationID: {}}) {
		t.Fatalf("protected IDs = %#v err=%v", protected, err)
	}
	for _, status := range []OpeningSessionStatus{
		OpeningStatusPaused, OpeningStatusRestartRequired,
	} {
		if err := store.SetSessionStatus(ctx, session.ID, status, now.Add(time.Minute)); err != nil {
			t.Fatal(err)
		}
		protected, err = store.ProtectedGenerationIDs(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, exists := protected[session.GenerationID]; !exists {
			t.Fatalf("status %q did not protect generation: %#v", status, protected)
		}
	}
	if err := store.SetSessionStatus(ctx, session.ID, OpeningStatusCompleted, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	protected, err = store.ProtectedGenerationIDs(ctx)
	if err != nil || len(protected) != 0 {
		t.Fatalf("protected completed IDs = %#v err=%v", protected, err)
	}
}
