package openings

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
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
			Position: PositionState{
				PositionID: "after-bc5", CurrentFEN: "fen-after-bc5",
				PlayedMoveIDs: []string{"white-e4", "black-e5"},
			},
			Attempt: &AttemptState{
				AttemptID: "attempt-1", PromptID: "recall-c3", StartedAt: now,
				HintLevel: 1, AlternativesTried: 1,
			},
		},
	}
}

func mustAttemptRecord(t *testing.T, attempt *AttemptState) AttemptRecord {
	t.Helper()
	record, err := attemptRecord(attempt)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func TestUserStoreJourneyRoundTrip(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	store := NewUserStore(openOpeningUserTestDB(t))
	want := CourseJourney{
		CourseID: "italian-white", CurrentLessonID: "giuoco-plan",
		PathLessonIDs: []string{"foundations", "giuoco-plan"},
		CreatedAt:     now, UpdatedAt: now,
	}
	if err := store.SaveJourney(ctx, want); err != nil {
		t.Fatal(err)
	}
	got, err := store.Journey(ctx, want.CourseID)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%#v want=%#v err=%v", got, want, err)
	}
}

func TestUserStoreJourneyReturnsInitializedCourse(t *testing.T) {
	store := NewUserStore(openOpeningUserTestDB(t))
	got, err := store.Journey(context.Background(), "italian-white")
	if err != nil {
		t.Fatal(err)
	}
	if got.CourseID != "italian-white" || got.PathLessonIDs == nil {
		t.Fatalf("journey = %#v", got)
	}
}

func TestCompletedOpeningNodeStaysCompleteWhenRequirementsGrow(t *testing.T) {
	ctx := context.Background()
	store := NewUserStore(openOpeningUserTestDB(t))
	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	err := store.RecordActivityProgress(ctx, ActivityProgressUpdate{
		CourseID: "italian-white", LessonID: "giuoco-plan",
		CompletedActivityID: "decision-c3", RequiredActivityIDs: []string{"decision-c3"}, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	progress, err := store.LessonProgress(ctx, "italian-white", "giuoco-plan", []string{"decision-c3", "new-required"})
	if err != nil || !progress.Completed {
		t.Fatalf("progress=%+v err=%v", progress, err)
	}
}

func TestUserStoreRecordsPartialActivityProgressInAuthoredOrder(t *testing.T) {
	ctx := context.Background()
	store := NewUserStore(openOpeningUserTestDB(t))
	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	required := []string{"concept", "decision", "recap"}
	for _, activityID := range []string{"decision", "concept", "decision"} {
		if err := store.RecordActivityProgress(ctx, ActivityProgressUpdate{
			CourseID: "italian-white", LessonID: "giuoco-plan",
			CompletedActivityID: activityID, RequiredActivityIDs: required, Now: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	progress, err := store.LessonProgress(ctx, "italian-white", "giuoco-plan", required)
	if err != nil {
		t.Fatal(err)
	}
	if progress.Completed || !reflect.DeepEqual(progress.CompletedActivityIDs, []string{"concept", "decision"}) {
		t.Fatalf("progress = %#v", progress)
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
	session.ActivityIndex = 3
	session.State.Position.CurrentFEN = "fen-after-c3"
	session.State.Position.PlayedMoveIDs = append(session.State.Position.PlayedMoveIDs, "white-c3")
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
	for _, status := range []OpeningSessionStatus{OpeningStatusActive, OpeningStatusPaused} {
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
	reviewSeed.State.Review = &ReviewCursor{
		PromptIDs: []string{"recall-c3", "recall-d3"}, Index: 1,
	}
	reviewSession, err := store.CreateSession(ctx, reviewSeed, now.Add(6*time.Minute))
	if err != nil || reviewSession.LessonID != "review" || reviewSession.Mode != OpeningModeReview {
		t.Fatalf("review session = %+v err=%v", reviewSession, err)
	}
}

func TestUserStoreRebasesSessionGenerationAndCheckpointExactly(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	store := NewUserStore(openOpeningUserTestDB(t))
	session, err := store.CreateSession(ctx, openingSessionSeed(now), now)
	if err != nil {
		t.Fatal(err)
	}
	previousGenerationID := session.GenerationID
	session.GenerationID = "generation-2"
	session.LessonID = "replacement-lesson"
	session.Status = OpeningStatusRestartRequired
	session.ActivityIndex = 2
	session.State.Attempt = nil
	session.State.Restart = &RestartCheckpoint{ActivityIndex: 1}
	if err := store.ApplyCourseRevision(ctx, CourseRevision{
		CourseID: session.CourseID, PromptFingerprints: map[string]string{},
		SessionRebase: &SessionRebase{
			PreviousGenerationID: previousGenerationID, Session: session,
		},
		Now: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.LoadSession(ctx, session.ID)
	if err != nil || !reflect.DeepEqual(loaded, session) {
		t.Fatalf("rebased session = %#v, want %#v, err=%v", loaded, session, err)
	}
	if err := store.ApplyCourseRevision(ctx, CourseRevision{
		CourseID: session.CourseID, PromptFingerprints: map[string]string{},
		SessionRebase: &SessionRebase{
			PreviousGenerationID: previousGenerationID, Session: session,
		},
		Now: now.Add(2 * time.Minute),
	}); err == nil {
		t.Fatal("rebase accepted a stale previous generation")
	}
}

func TestApplyCourseRevisionRollsBackSessionWhenReviewArchiveFails(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	db := openOpeningUserTestDB(t)
	store := NewUserStore(db)
	session, err := store.CreateSession(ctx, openingSessionSeed(now), now)
	if err != nil {
		t.Fatal(err)
	}
	beforeJourney := CourseJourney{
		CourseID: session.CourseID, CurrentLessonID: session.LessonID,
		PathLessonIDs: []string{session.LessonID},
		CreatedAt:     now, UpdatedAt: now,
	}
	if err := store.SaveJourney(ctx, beforeJourney); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO opening_review_state(
		 course_id, prompt_id, semantic_fingerprint, due_at, interval_index,
		 successful_reviews, last_outcome, status
		) VALUES (?, 'retired', 'old', ?, 2, 3, 'clean', 'active')`,
		session.CourseID,
		now.UnixMilli(),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TRIGGER fail_opening_review_archive
		BEFORE UPDATE OF status ON opening_review_state
		WHEN NEW.status = 'archived'
		BEGIN
		  SELECT RAISE(ABORT, 'forced review archive failure');
		END`); err != nil {
		t.Fatal(err)
	}

	previousGenerationID := session.GenerationID
	session.GenerationID = "generation-2"
	updatedJourney := beforeJourney
	updatedJourney.CurrentLessonID = "replacement-lesson"
	updatedJourney.PathLessonIDs = []string{"replacement-lesson"}
	updatedJourney.UpdatedAt = now.Add(time.Minute)
	err = store.ApplyCourseRevision(ctx, CourseRevision{
		CourseID:           session.CourseID,
		PromptFingerprints: map[string]string{},
		SessionRebase: &SessionRebase{
			PreviousGenerationID: previousGenerationID,
			Session:              session,
		},
		Journey: &updatedJourney,
		Now:     now.Add(time.Minute),
	})
	if err == nil || !strings.Contains(err.Error(), "forced review archive failure") {
		t.Fatalf("ApplyCourseRevision() error = %v", err)
	}
	loaded, err := store.LoadSession(ctx, session.ID)
	if err != nil || loaded.GenerationID != previousGenerationID {
		t.Fatalf("session after rollback = %+v, err=%v", loaded, err)
	}
	afterJourney, err := store.Journey(ctx, session.CourseID)
	if err != nil || !reflect.DeepEqual(afterJourney, beforeJourney) {
		t.Fatalf("journey after rollback = %+v, want %+v, err=%v", afterJourney, beforeJourney, err)
	}
	var status string
	if err := db.QueryRow(
		`SELECT status FROM opening_review_state WHERE course_id = ? AND prompt_id = 'retired'`,
		session.CourseID,
	).Scan(&status); err != nil || status != "active" {
		t.Fatalf("review status after rollback = %q, err=%v", status, err)
	}
}

func TestApplyCourseRevisionKeepsCompletedLessonStickyWhenActivitiesGrow(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	store := NewUserStore(openOpeningUserTestDB(t))
	if err := store.RecordActivityProgress(ctx, ActivityProgressUpdate{
		CourseID: "italian-white", LessonID: "legacy-lesson",
		CompletedActivityID: "legacy-decision", RequiredActivityIDs: []string{"legacy-decision"}, Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.ApplyCourseRevision(ctx, CourseRevision{
		CourseID: "italian-white", PromptFingerprints: map[string]string{}, Now: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	progress, err := store.LessonProgress(
		ctx, "italian-white", "legacy-lesson", []string{"legacy-decision", "new-required-concept"},
	)
	if err != nil || !progress.Completed {
		t.Fatalf("progress=%+v err=%v", progress, err)
	}
}

func TestUserStoreCourseRevisionArchivesOnlyRemovedOrChangedPrompts(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	db := openOpeningUserTestDB(t)
	store := NewUserStore(db)
	for _, row := range []struct{ promptID, fingerprint string }{
		{"unchanged", "same"}, {"changed", "old"}, {"removed", "gone"},
	} {
		if _, err := db.Exec(
			`INSERT INTO opening_review_state(
			 course_id, prompt_id, semantic_fingerprint, due_at, interval_index,
			 successful_reviews, last_outcome, status
			) VALUES ('italian-white', ?, ?, ?, 2, 3, 'clean', 'active')`,
			row.promptID, row.fingerprint, now.UnixMilli(),
		); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.ApplyCourseRevision(ctx, CourseRevision{
		CourseID: "italian-white", Now: now.Add(time.Minute),
		PromptFingerprints: map[string]string{
			"unchanged": "same", "changed": "new",
		},
	}); err != nil {
		t.Fatal(err)
	}
	for promptID, want := range map[string]string{
		"unchanged": "active", "changed": "archived", "removed": "archived",
	} {
		var status string
		if err := db.QueryRow(
			`SELECT status FROM opening_review_state WHERE course_id = 'italian-white' AND prompt_id = ?`,
			promptID,
		).Scan(&status); err != nil || status != want {
			t.Fatalf("prompt %q status = %q, want %q, err=%v", promptID, status, want, err)
		}
	}
}

func TestOpeningLessonProgressProjectsStableActivityIDsAndKeepsCompletion(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	store := NewUserStore(openOpeningUserTestDB(t))
	session, err := store.CreateSession(ctx, openingSessionSeed(now), now)
	if err != nil {
		t.Fatal(err)
	}
	attempt := mustAttemptRecord(t, session.State.Attempt)
	session.Status = OpeningStatusCompleted
	session.State.Attempt = nil
	completed := []string{"explain", "watch", "try", "branch", "recall"}
	if err := store.CompletePrompt(ctx, PromptCompletion{
		Session: session, Attempt: attempt, SemanticFingerprint: "semantic-v1",
		Outcome: ReviewClean, CompletedActivityIDs: completed,
	}, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	progress, err := store.LessonProgress(ctx, "italian-white", "giuoco-c3", completed)
	if err != nil || !progress.Completed || progress.CompletedActivities != 5 || progress.TotalActivities != 5 {
		t.Fatalf("original progress = %+v err=%v", progress, err)
	}
	updated := []string{"explain", "watch", "inserted", "try", "branch", "recall"}
	progress, err = store.LessonProgress(ctx, "italian-white", "giuoco-c3", updated)
	if err != nil || !progress.Completed || progress.CompletedActivities != 5 || progress.TotalActivities != 6 {
		t.Fatalf("updated progress = %+v err=%v", progress, err)
	}
	shortened := []string{"explain", "watch", "try", "branch"}
	progress, err = store.LessonProgress(ctx, "italian-white", "giuoco-c3", shortened)
	if err != nil || !progress.Completed || progress.CompletedActivities != 4 || progress.TotalActivities != 4 {
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
	session.ActivityIndex = 5
	session.Status = OpeningStatusCompleted
	attempt := mustAttemptRecord(t, session.State.Attempt)
	session.State.Attempt = nil
	completedSteps := []string{"explain", "watch", "try", "branch", "recall"}
	if err := store.CompletePrompt(ctx, PromptCompletion{
		Session: session, Attempt: attempt,
		SemanticFingerprint: "semantic-v1", Outcome: ReviewClean,
		CompletedActivityIDs: completedSteps,
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
		attempt.PromptID,
	).Scan(&promptFingerprint, &promptOutcome); err != nil {
		t.Fatal(err)
	}
	if promptFingerprint != "semantic-v1" || promptOutcome != ReviewClean {
		t.Fatalf("prompt progress = fingerprint %q outcome %q", promptFingerprint, promptOutcome)
	}
	review, err := store.Review(ctx, session.CourseID, attempt.PromptID)
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
	if err != nil || loaded.Status != OpeningStatusCompleted || loaded.ActivityIndex != 5 {
		t.Fatalf("completed session = %+v err=%v", loaded, err)
	}

	for index, days := range []int{3, 7, 21, 60} {
		nextNow := review.DueAt
		attempt.AttemptID = "attempt-clean-" + string(rune('a'+index))
		attempt.StartedAt = nextNow.Add(-time.Minute)
		if err := store.CompletePrompt(ctx, PromptCompletion{
			Session: session, Attempt: attempt,
			SemanticFingerprint: "semantic-v1", Outcome: ReviewClean,
		}, nextNow); err != nil {
			t.Fatal(err)
		}
		review, err = store.Review(ctx, session.CourseID, attempt.PromptID)
		if err != nil || review.DueAt.Sub(nextNow) != time.Duration(days)*24*time.Hour {
			t.Fatalf("clean review %d = %+v err=%v", index, review, err)
		}
	}
	attempt.AttemptID = "attempt-hinted"
	attempt.StartedAt = review.DueAt.Add(-time.Minute)
	attempt.HintsUsed = 1
	if err := store.CompletePrompt(ctx, PromptCompletion{
		Session: session, Attempt: attempt,
		SemanticFingerprint: "semantic-v1", Outcome: ReviewHinted,
	}, review.DueAt); err != nil {
		t.Fatal(err)
	}
	reset, err := store.Review(ctx, session.CourseID, attempt.PromptID)
	if err != nil || reset.IntervalIndex != 0 || reset.SuccessfulReviews != 0 ||
		reset.DueAt.Sub(review.DueAt) != 24*time.Hour {
		t.Fatalf("hinted reset = %+v err=%v", reset, err)
	}
	due, err := store.DueReviews(ctx, session.CourseID, reset.DueAt, 10)
	if err != nil || len(due) != 1 || due[0].PromptID != attempt.PromptID {
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
		Session: missing, Attempt: mustAttemptRecord(t, seed.State.Attempt),
		SemanticFingerprint: "semantic-v1", Outcome: ReviewClean,
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
	if err := store.SetSessionStatus(ctx, session.ID, OpeningStatusPaused, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	for _, status := range []OpeningSessionStatus{OpeningStatusPaused, OpeningStatusRestartRequired} {
		if status == OpeningStatusRestartRequired {
			session, err = store.LoadSession(ctx, session.ID)
			if err != nil {
				t.Fatal(err)
			}
			session.Status = status
			session.State.Attempt = nil
			if err := store.SaveSession(ctx, session, now.Add(time.Minute)); err != nil {
				t.Fatal(err)
			}
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
