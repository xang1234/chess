package openings

import (
	"context"
	"reflect"
	"testing"
	"time"

	"chess-trainer/internal/chessrules"
)

func TestOpeningServiceCompletesActivitiesAndReturnsCheckpoint(t *testing.T) {
	ctx := context.Background()
	fixture := newTreeServiceFixture(t)
	started, err := fixture.service.StartLesson(ctx, fixture.compiled.Pack.CourseID, "giuoco-plan")
	if err != nil || started.Current == nil || started.Current.Kind != ActivityConcept {
		t.Fatalf("started=%+v err=%v", started, err)
	}
	if started.Current.ActivityID != "giuoco-concept" || started.Current.ActivityNumber != 1 ||
		started.Current.ActivityTotal != 3 || started.Current.RequiredIdeas != 3 ||
		len(started.Current.MovesToHere) != 6 {
		t.Fatalf("started activity=%+v", started.Current)
	}
	movesToHere := make([]string, len(started.Current.MovesToHere))
	for index, move := range started.Current.MovesToHere {
		movesToHere[index] = move.UCI
	}
	if !reflect.DeepEqual(movesToHere, []string{"e2e4", "e7e5", "g1f3", "b8c6", "f1c4", "f8c5"}) {
		t.Fatalf("moves to here=%v", movesToHere)
	}
	concept, err := fixture.service.AdvanceActivity(ctx, started.SessionID)
	if err != nil || concept.Session.Current == nil || concept.Session.Current.Kind != ActivityDecision {
		t.Fatalf("concept=%+v err=%v", concept, err)
	}
	if !concept.ActivityCompleted || concept.Session.Current.CompletedIdeas != 1 {
		t.Fatalf("concept=%+v", concept)
	}
	decision, err := fixture.service.PlayMove(ctx, started.SessionID, "c2c3")
	if err != nil || decision.Session.Current == nil || decision.Session.Current.Kind != ActivityRecap {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	if decision.Feedback != FeedbackExpected || len(decision.AppliedMoves) != 1 {
		t.Fatalf("decision=%+v", decision)
	}
	done, err := fixture.service.AdvanceActivity(ctx, started.SessionID)
	if err != nil || done.Checkpoint == nil || done.Checkpoint.RecommendedLessonID != "two-knights-plan" {
		t.Fatalf("done=%+v err=%v", done, err)
	}
	if done.Session.Status != OpeningStatusCompleted || done.Session.Current != nil ||
		done.Checkpoint.CompletedLessonID != "giuoco-plan" ||
		!reflect.DeepEqual(done.Checkpoint.AvailableLessonIDs, []string{"two-knights-plan"}) {
		t.Fatalf("done=%+v", done)
	}
	journey, err := fixture.store.Journey(ctx, fixture.compiled.Pack.CourseID)
	if err != nil || journey.CurrentLessonID != "giuoco-plan" ||
		!reflect.DeepEqual(journey.PathLessonIDs, []string{"giuoco-plan"}) {
		t.Fatalf("journey=%+v err=%v", journey, err)
	}
}

func TestOpeningServiceResumesExactActivityWithoutReplayingConcept(t *testing.T) {
	ctx := context.Background()
	fixture := newTreeServiceFixture(t)
	started, err := fixture.service.StartLesson(ctx, fixture.compiled.Pack.CourseID, "giuoco-plan")
	if err != nil {
		t.Fatal(err)
	}
	advanced, err := fixture.service.AdvanceActivity(ctx, started.SessionID)
	if err != nil || advanced.Session.Current == nil || advanced.Session.Current.ActivityID != "giuoco-c3-decision" {
		t.Fatalf("advanced=%+v err=%v", advanced, err)
	}
	if err := fixture.service.Pause(ctx, started.SessionID); err != nil {
		t.Fatal(err)
	}
	recreated := NewService(fixture.service.catalog, fixture.store, chessrules.Rules{}, "")
	recreated.now = fixture.service.now
	resumed, err := recreated.Resume(ctx)
	if err != nil || resumed == nil || resumed.Current == nil {
		t.Fatalf("resumed=%+v err=%v", resumed, err)
	}
	if resumed.Current.ActivityID != "giuoco-c3-decision" ||
		resumed.Current.CurrentFEN != fixture.compiled.Positions["after-bc5"].FEN ||
		resumed.Current.CompletedIdeas != 1 {
		t.Fatalf("resumed activity=%+v", resumed.Current)
	}
	progress, err := fixture.store.LessonProgress(
		ctx, fixture.compiled.Pack.CourseID, "giuoco-plan",
		RequiredActivityIDs(fixture.compiled.Lessons["giuoco-plan"]),
	)
	if err != nil || !reflect.DeepEqual(progress.CompletedActivityIDs, []string{"giuoco-concept"}) {
		t.Fatalf("progress=%+v err=%v", progress, err)
	}
}

func TestOpeningServiceExposesOptionalReferenceOutsideRequiredCursor(t *testing.T) {
	pack := decodeTreePack(t)
	pack.Lessons[0].Activities = append(pack.Lessons[0].Activities, LessonActivity{
		ActivityID: "giuoco-reference", Kind: ActivityReference, Title: "Deeper source analysis",
		Instruction: "Inspect the optional analytical note.", Required: false,
		PositionID: "after-bc5", NoteIDs: []string{"italian-overview"}, MoveIDs: []string{},
	})
	compiled, err := Compile(pack, chessrules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	catalog := NewSQLiteCatalog(openCourseCatalogTestDB(t))
	result, err := catalog.Replace(ctx, compiled, "/private/tree-reference.ctcourse", "sha-tree-reference")
	if err != nil {
		t.Fatal(err)
	}
	store := NewUserStore(openOpeningUserTestDB(t))
	service := NewService(catalog, store, chessrules.Rules{}, "")
	service.now = func() time.Time { return time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC) }
	started, err := service.StartLesson(ctx, compiled.Pack.CourseID, "giuoco-plan")
	if err != nil {
		t.Fatal(err)
	}
	if started.GenerationID != result.GenerationID || started.Current == nil ||
		started.Current.ActivityTotal != 3 || started.Current.RequiredIdeas != 3 ||
		len(started.Current.ReferenceSections) != 1 ||
		started.Current.ReferenceSections[0].ActivityID != "giuoco-reference" {
		t.Fatalf("started=%+v", started)
	}
	if _, err := service.AdvanceActivity(ctx, started.SessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.PlayMove(ctx, started.SessionID, "c2c3"); err != nil {
		t.Fatal(err)
	}
	done, err := service.AdvanceActivity(ctx, started.SessionID)
	if err != nil || done.Checkpoint == nil {
		t.Fatalf("done=%+v err=%v", done, err)
	}
	progress, err := store.LessonProgress(ctx, compiled.Pack.CourseID, "giuoco-plan", RequiredActivityIDs(compiled.Lessons["giuoco-plan"]))
	if err != nil || progress.TotalActivities != 3 || progress.CompletedActivities != 3 ||
		!reflect.DeepEqual(progress.CompletedActivityIDs, []string{"giuoco-concept", "giuoco-c3-decision", "giuoco-recap"}) {
		t.Fatalf("progress=%+v err=%v", progress, err)
	}
}

func TestActivityComparisonLinesUseSANInsteadOfInternalMoveIDs(t *testing.T) {
	course, err := Compile(decodeTreePack(t), chessrules.Rules{})
	if err != nil {
		t.Fatal(err)
	}

	lines := activityComparisonLines(course, []ActivityLine{{
		Label:   "Develop the Italian",
		MoveIDs: []string{"white-e4", "black-e5", "white-nf3"},
	}})

	want := []OpeningActivityLine{{Label: "Develop the Italian", Moves: []string{"e4", "e5", "Nf3"}}}
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("comparison lines = %#v, want %#v", lines, want)
	}
}

func TestOpeningLessonStartRollsBackSessionWhenJourneyWriteFails(t *testing.T) {
	fixture := newTreeServiceFixture(t)
	ctx := context.Background()
	if _, err := fixture.store.db.Exec(`CREATE TRIGGER fail_opening_journey_insert
		BEFORE INSERT ON opening_course_journeys
		BEGIN SELECT RAISE(ABORT, 'forced journey failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.StartLesson(ctx, fixture.compiled.Pack.CourseID, "giuoco-plan"); err == nil {
		t.Fatal("StartLesson() unexpectedly succeeded")
	}
	var sessions, journeys int
	if err := fixture.store.db.QueryRow(`SELECT COUNT(*) FROM opening_sessions`).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if err := fixture.store.db.QueryRow(`SELECT COUNT(*) FROM opening_course_journeys`).Scan(&journeys); err != nil {
		t.Fatal(err)
	}
	if sessions != 0 || journeys != 0 {
		t.Fatalf("sessions=%d journeys=%d", sessions, journeys)
	}
}

func TestOpeningServiceCanStartAnyVisibleNodeWhileAnotherLessonIsActive(t *testing.T) {
	fixture := newTreeServiceFixture(t)
	ctx := context.Background()
	first, err := fixture.service.StartLesson(ctx, fixture.compiled.Pack.CourseID, "giuoco-plan")
	if err != nil {
		t.Fatal(err)
	}
	second, err := fixture.service.StartLesson(ctx, fixture.compiled.Pack.CourseID, "two-knights-plan")
	if err != nil {
		t.Fatal(err)
	}
	retired, err := fixture.store.LoadSession(ctx, first.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if retired.Status != OpeningStatusCompleted || second.Status != OpeningStatusActive ||
		second.Current == nil || second.Current.ActivityID != "two-knights-d3-decision" {
		t.Fatalf("retired=%+v second=%+v", retired, second)
	}
	journey, err := fixture.store.Journey(ctx, fixture.compiled.Pack.CourseID)
	if err != nil || journey.CurrentLessonID != second.LessonID ||
		!reflect.DeepEqual(journey.PathLessonIDs, []string{"giuoco-plan", "two-knights-plan"}) {
		t.Fatalf("journey=%+v err=%v", journey, err)
	}
}

func TestOpeningActivityCompletionRollsBackSessionProgressAndJourney(t *testing.T) {
	fixture := newTreeServiceFixture(t)
	ctx := context.Background()
	started, err := fixture.service.StartLesson(ctx, fixture.compiled.Pack.CourseID, "giuoco-plan")
	if err != nil {
		t.Fatal(err)
	}
	beforeSession, err := fixture.store.LoadSession(ctx, started.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	beforeJourney, err := fixture.store.Journey(ctx, fixture.compiled.Pack.CourseID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.store.db.Exec(`CREATE TRIGGER fail_opening_journey_update
		BEFORE UPDATE ON opening_course_journeys
		BEGIN SELECT RAISE(ABORT, 'forced journey update failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.AdvanceActivity(ctx, started.SessionID); err == nil {
		t.Fatal("AdvanceActivity() unexpectedly succeeded")
	}
	afterSession, err := fixture.store.LoadSession(ctx, started.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	afterJourney, err := fixture.store.Journey(ctx, fixture.compiled.Pack.CourseID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(afterSession, beforeSession) || !reflect.DeepEqual(afterJourney, beforeJourney) {
		t.Fatalf("session before=%+v after=%+v journey before=%+v after=%+v", beforeSession, afterSession, beforeJourney, afterJourney)
	}
	var progressRows int
	if err := fixture.store.db.QueryRow(`SELECT COUNT(*) FROM opening_lesson_progress`).Scan(&progressRows); err != nil {
		t.Fatal(err)
	}
	if progressRows != 0 {
		t.Fatalf("progress rows=%d", progressRows)
	}
}

func TestOpeningDecisionCompletionRollsBackAttemptReviewProgressAndJourney(t *testing.T) {
	fixture := newTreeServiceFixture(t)
	ctx := context.Background()
	started, err := fixture.service.StartLesson(ctx, fixture.compiled.Pack.CourseID, "giuoco-plan")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.AdvanceActivity(ctx, started.SessionID); err != nil {
		t.Fatal(err)
	}
	beforeSession, err := fixture.store.LoadSession(ctx, started.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	beforeJourney, err := fixture.store.Journey(ctx, fixture.compiled.Pack.CourseID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.store.db.Exec(`CREATE TRIGGER fail_opening_decision_progress
		BEFORE UPDATE ON opening_lesson_progress
		BEGIN SELECT RAISE(ABORT, 'forced decision progress failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.PlayMove(ctx, started.SessionID, "c2c3"); err == nil {
		t.Fatal("PlayMove() unexpectedly succeeded")
	}
	afterSession, err := fixture.store.LoadSession(ctx, started.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	afterJourney, err := fixture.store.Journey(ctx, fixture.compiled.Pack.CourseID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(afterSession, beforeSession) || !reflect.DeepEqual(afterJourney, beforeJourney) {
		t.Fatalf("session before=%+v after=%+v journey before=%+v after=%+v", beforeSession, afterSession, beforeJourney, afterJourney)
	}
	for _, table := range []string{"opening_attempts", "opening_prompt_progress", "opening_review_state"} {
		var rows int
		if err := fixture.store.db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&rows); err != nil {
			t.Fatal(err)
		}
		if rows != 0 {
			t.Fatalf("%s rows=%d", table, rows)
		}
	}
	progress, err := fixture.store.LessonProgress(
		ctx, fixture.compiled.Pack.CourseID, "giuoco-plan",
		RequiredActivityIDs(fixture.compiled.Lessons["giuoco-plan"]),
	)
	if err != nil || !reflect.DeepEqual(progress.CompletedActivityIDs, []string{"giuoco-concept"}) {
		t.Fatalf("progress=%+v err=%v", progress, err)
	}
}
