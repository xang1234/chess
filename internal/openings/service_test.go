package openings

import (
	"context"
	"database/sql"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"chess-trainer/internal/chessrules"
)

type openingServiceFixture struct {
	service  *Service
	catalog  *SQLiteCatalog
	store    *UserStore
	userDB   *sql.DB
	compiled CompiledCourse
	result   ReplaceResult
	now      time.Time
}

func newOpeningServiceFixture(t *testing.T) openingServiceFixture {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	compiled := compileMiniCourse(t)
	catalog := NewSQLiteCatalog(openCourseCatalogTestDB(t))
	result, err := catalog.Replace(ctx, compiled, "/private/mini.ctcourse", "sha-mini")
	if err != nil {
		t.Fatal(err)
	}
	userDB := openOpeningUserTestDB(t)
	store := NewUserStore(userDB)
	service := NewService(catalog, store, chessrules.Rules{}, "Private course storage was rebuilt.")
	service.now = func() time.Time { return now }
	return openingServiceFixture{
		service: service, catalog: catalog, store: store, userDB: userDB,
		compiled: compiled, result: result, now: now,
	}
}

func TestOpeningServiceHomeFiltersCourseAtSelectedDepth(t *testing.T) {
	ctx := context.Background()
	fixture := newOpeningServiceFixture(t)

	home, err := fixture.service.Home(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if home.Notice != "Private course storage was rebuilt." || len(home.Courses) != 1 {
		t.Fatalf("home = %+v", home)
	}
	course := home.Courses[0]
	if course.Depth != DepthReference || course.TotalLessons != 1 || len(course.Chapters) != 3 {
		t.Fatalf("reference course = %+v", course)
	}
	if course.NextLessonID != "giuoco-c3" || course.NextLessonTitle == "" {
		t.Fatalf("next lesson = %q %q", course.NextLessonID, course.NextLessonTitle)
	}

	if err := fixture.service.SetDepth(ctx, course.CourseID, DepthQuick); err != nil {
		t.Fatal(err)
	}
	home, err = fixture.service.Home(ctx)
	if err != nil {
		t.Fatal(err)
	}
	course = home.Courses[0]
	if course.Depth != DepthQuick || len(course.Chapters) != 2 {
		t.Fatalf("quick course = %+v", course)
	}
	for _, chapter := range course.Chapters {
		if chapter.ChapterID == "alternatives" {
			t.Fatal("reference-only chapter leaked into quick depth")
		}
	}
}

func TestOpeningServiceHomeIsReadOnlyForStaleReviews(t *testing.T) {
	ctx := context.Background()
	fixture := newOpeningServiceFixture(t)
	if _, err := fixture.userDB.Exec(
		`INSERT INTO opening_review_state(
		 course_id, prompt_id, semantic_fingerprint, due_at, interval_index,
		 successful_reviews, last_outcome, status
		) VALUES (?, 'retired', 'old', ?, 2, 3, 'clean', 'active')`,
		fixture.compiled.Pack.CourseID,
		fixture.now.Add(-time.Hour).UnixMilli(),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.userDB.Exec(`CREATE TRIGGER fail_opening_review_archive
		BEFORE UPDATE OF status ON opening_review_state
		WHEN NEW.status = 'archived'
		BEGIN
		  SELECT RAISE(ABORT, 'home attempted a review write');
		END`); err != nil {
		t.Fatal(err)
	}

	home, err := fixture.service.Home(ctx)
	if err != nil {
		t.Fatalf("Home() performed a write: %v", err)
	}
	if len(home.Courses) != 1 || home.Courses[0].DueReviews != 0 {
		t.Fatalf("home stale review projection = %+v", home)
	}
}

func TestOpeningServiceSequencesLessonFeedbackHintsAndCompletion(t *testing.T) {
	ctx := context.Background()
	fixture := newOpeningServiceFixture(t)
	if _, err := fixture.userDB.Exec(
		`INSERT INTO profile(id, learner_rating, session_size, created_at, updated_at)
		 VALUES (1, 1432, 10, ?, ?)`, fixture.now.UnixMilli(), fixture.now.UnixMilli(),
	); err != nil {
		t.Fatal(err)
	}

	started, err := fixture.service.StartLesson(ctx, "synthetic-italian", "giuoco-c3")
	if err != nil {
		t.Fatal(err)
	}
	assertOpeningActivity(t, started, ActivityConcept, "explain-plan")
	if started.Current.Orientation != PerspectiveWhite || len(started.Current.LegalMoves) != 0 {
		t.Fatalf("explain step = %+v", started.Current)
	}

	advanced, err := fixture.service.Advance(ctx, started.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	assertOpeningActivity(t, advanced.Session, ActivityDemonstration, "watch-setup")
	if len(advanced.AppliedMoves) != 0 {
		t.Fatalf("explain advance frames = %+v", advanced.AppliedMoves)
	}

	advanced, err = fixture.service.Advance(ctx, started.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	assertOpeningActivity(t, advanced.Session, ActivityDecision, "try-c3")
	if len(advanced.AppliedMoves) != 6 || advanced.FinalFEN != fixture.compiled.Positions["after-bc5"].FEN {
		t.Fatalf("watch advance = %+v", advanced)
	}
	if !slices.Contains(advanced.Session.Current.LegalMoves, "c2c3") ||
		slices.Contains(advanced.Session.Current.LegalMoves, "white-c3") {
		t.Fatalf("prompt legal moves = %v", advanced.Session.Current.LegalMoves)
	}

	alternative, err := fixture.service.PlayMove(ctx, started.SessionID, "b2b4")
	if err != nil {
		t.Fatal(err)
	}
	if alternative.Feedback != FeedbackAlternative || alternative.StepCompleted ||
		alternative.Session.Current.CurrentFEN != fixture.compiled.Positions["after-bc5"].FEN ||
		len(alternative.AppliedMoves) != 0 || alternative.FinalFEN != "" {
		t.Fatalf("alternative result = %+v", alternative)
	}
	if !strings.Contains(strings.ToLower(alternative.Message), "course alternative") {
		t.Fatalf("alternative message = %q", alternative.Message)
	}
	stored, err := fixture.store.LoadSession(ctx, started.SessionID)
	if err != nil || stored.State.Attempt == nil ||
		stored.State.Attempt.AlternativesTried != 1 || stored.State.Attempt.IncorrectMoves != 0 {
		t.Fatalf("alternative state = %+v err=%v", stored.State, err)
	}

	offCourse, err := fixture.service.PlayMove(ctx, started.SessionID, "d2d3")
	if err != nil {
		t.Fatal(err)
	}
	if offCourse.Feedback != FeedbackOffCourse || offCourse.StepCompleted ||
		offCourse.Message != "That move is playable, but this lesson is practicing c3." {
		t.Fatalf("off-course result = %+v", offCourse)
	}
	beforeIllegal, err := fixture.store.LoadSession(ctx, started.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.PlayMove(ctx, started.SessionID, "e2e5"); err == nil {
		t.Fatal("illegal UCI move was accepted")
	}
	afterIllegal, err := fixture.store.LoadSession(ctx, started.SessionID)
	if err != nil || !reflect.DeepEqual(afterIllegal.State, beforeIllegal.State) {
		t.Fatalf("illegal move persisted state: before=%+v after=%+v err=%v", beforeIllegal.State, afterIllegal.State, err)
	}

	for level := 1; level <= 4; level++ {
		hint, err := fixture.service.UseHint(ctx, started.SessionID)
		if err != nil {
			t.Fatal(err)
		}
		if hint.Level != level {
			t.Fatalf("hint level = %d, want %d", hint.Level, level)
		}
		switch level {
		case 1:
			if !strings.Contains(hint.Text, "Develop quickly") {
				t.Fatalf("plan hint = %+v", hint)
			}
		case 2:
			if hint.SourceSquare != "c2" || hint.TargetSquare != "" {
				t.Fatalf("source hint = %+v", hint)
			}
		case 3:
			if hint.SourceSquare != "c2" || hint.TargetSquare != "c3" {
				t.Fatalf("target hint = %+v", hint)
			}
		case 4:
			if !hint.CanReveal || hint.Text != "Show the course move." {
				t.Fatalf("reveal hint = %+v", hint)
			}
		}
	}
	revealed, err := fixture.service.Reveal(ctx, started.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if !revealed.StepCompleted || revealed.Feedback != FeedbackExpected || len(revealed.AppliedMoves) != 1 {
		t.Fatalf("reveal result = %+v", revealed)
	}
	assertOpeningActivity(t, revealed.Session, ActivityDecision, "branch-giuoco")
	var attemptIncorrect, attemptAlternatives, attemptHints, attemptRevealed int
	if err := fixture.userDB.QueryRow(
		`SELECT incorrect_moves, alternatives_tried, hints_used, revealed
		 FROM opening_attempts ORDER BY started_at, attempt_id LIMIT 1`,
	).Scan(&attemptIncorrect, &attemptAlternatives, &attemptHints, &attemptRevealed); err != nil {
		t.Fatal(err)
	}
	if attemptIncorrect != 1 || attemptAlternatives != 1 || attemptHints != 4 || attemptRevealed != 1 {
		t.Fatalf("completed attempt metrics = %d %d %d %d", attemptIncorrect, attemptAlternatives, attemptHints, attemptRevealed)
	}
	nextState, err := fixture.store.LoadSession(ctx, started.SessionID)
	if err != nil || nextState.State.Attempt == nil ||
		nextState.State.Attempt.IncorrectMoves != 0 || nextState.State.Attempt.AlternativesTried != 0 ||
		nextState.State.Attempt.HintsUsed != 0 || nextState.State.Attempt.Revealed {
		t.Fatalf("next attempt state = %+v err=%v", nextState.State, err)
	}

	expected, err := fixture.service.PlayMove(ctx, started.SessionID, "c2c3")
	if err != nil {
		t.Fatal(err)
	}
	if expected.Feedback != FeedbackExpected || expected.FinalFEN != fixture.compiled.Positions["after-c3"].FEN {
		t.Fatalf("expected branch result = %+v", expected)
	}
	assertOpeningActivity(t, expected.Session, ActivityDecision, "recall-c3-step")

	if err := fixture.service.Pause(ctx, started.SessionID); err != nil {
		t.Fatal(err)
	}
	paused, err := fixture.store.LoadSession(ctx, started.SessionID)
	if err != nil || paused.Status != OpeningStatusPaused {
		t.Fatalf("paused = %+v err=%v", paused, err)
	}
	resumed, err := fixture.service.Resume(ctx)
	if err != nil || resumed == nil {
		t.Fatalf("resume = %+v err=%v", resumed, err)
	}
	assertOpeningActivity(t, *resumed, ActivityDecision, "recall-c3-step")

	completed, err := fixture.service.PlayMove(ctx, started.SessionID, "c2c3")
	if err != nil {
		t.Fatal(err)
	}
	if completed.Session.Status != OpeningStatusCompleted || completed.Session.Current != nil ||
		completed.Session.Summary != nil || completed.Checkpoint == nil {
		t.Fatalf("completed session = %+v", completed.Session)
	}
	if completed.Checkpoint.CompletedLessonID != "giuoco-c3" {
		t.Fatalf("checkpoint = %+v", completed.Checkpoint)
	}
	var rating float64
	if err := fixture.userDB.QueryRow(`SELECT learner_rating FROM profile WHERE id = 1`).Scan(&rating); err != nil || rating != 1432 {
		t.Fatalf("puzzle rating = %v err=%v", rating, err)
	}
}

func TestOpeningServiceStartReviewPersistsOrderedQueue(t *testing.T) {
	ctx := context.Background()
	fixture := newOpeningServiceFixture(t)
	due := fixture.now.Add(-time.Hour).UnixMilli()
	for _, promptID := range []string{"recall-d3", "recall-c3"} {
		prompt := fixture.compiled.Prompts[promptID]
		if _, err := fixture.userDB.Exec(
			`INSERT INTO opening_review_state(
			 course_id, prompt_id, semantic_fingerprint, due_at, interval_index,
			 successful_reviews, last_outcome, status
			) VALUES (?, ?, ?, ?, 0, 1, 'clean', 'active')`,
			fixture.compiled.Pack.CourseID, promptID, prompt.SemanticFingerprint, due,
		); err != nil {
			t.Fatal(err)
		}
	}

	review, err := fixture.service.StartReview(ctx, fixture.compiled.Pack.CourseID)
	if err != nil {
		t.Fatal(err)
	}
	if review.Mode != OpeningModeReview || review.Current == nil || review.Current.PositionID != "after-bc5" {
		t.Fatalf("review start = %+v", review)
	}
	stored, err := fixture.store.LoadSession(ctx, review.SessionID)
	if err != nil || stored.State.Review == nil ||
		!slices.Equal(stored.State.Review.PromptIDs, []string{"recall-c3", "recall-d3"}) {
		t.Fatalf("review queue = %+v err=%v", stored.State.Review, err)
	}

	first, err := fixture.service.PlayMove(ctx, review.SessionID, "c2c3")
	if err != nil {
		t.Fatal(err)
	}
	if first.Session.Current == nil || first.Session.Current.PositionID != "after-nf6" {
		t.Fatalf("second review prompt = %+v", first.Session)
	}
	second, err := fixture.service.PlayMove(ctx, review.SessionID, "d2d3")
	if err != nil {
		t.Fatal(err)
	}
	if second.Session.Status != OpeningStatusCompleted || second.Session.Summary == nil || second.Session.Summary.TotalPrompts != 2 {
		t.Fatalf("review completion = %+v", second.Session)
	}
}

func assertOpeningActivity(t *testing.T, session OpeningSessionView, kind ActivityKind, activityID string) {
	t.Helper()
	if session.Status != OpeningStatusActive || session.Current == nil ||
		session.Current.Kind != kind || session.Current.ActivityID != activityID {
		t.Fatalf("session current = %+v, want %s %s", session, kind, activityID)
	}
}
