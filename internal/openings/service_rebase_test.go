package openings

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"testing"

	"chess-trainer/internal/chessrules"
)

func pauseMiniLessonAtRecall(t *testing.T, fixture openingServiceFixture) OpeningSessionView {
	t.Helper()
	ctx := context.Background()
	started, err := fixture.service.StartLesson(ctx, fixture.compiled.Pack.CourseID, "giuoco-c3")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.Advance(ctx, started.SessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.Advance(ctx, started.SessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.PlayMove(ctx, started.SessionID, "c2c3"); err != nil {
		t.Fatal(err)
	}
	branch, err := fixture.service.PlayMove(ctx, started.SessionID, "c2c3")
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.service.Pause(ctx, started.SessionID); err != nil {
		t.Fatal(err)
	}
	return branch.Session
}

func pauseTreeLessonAtDecision(t *testing.T, fixture treeServiceFixture) OpeningSessionView {
	t.Helper()
	ctx := context.Background()
	started, err := fixture.service.StartLesson(ctx, fixture.compiled.Pack.CourseID, "giuoco-plan")
	if err != nil {
		t.Fatal(err)
	}
	advanced, err := fixture.service.AdvanceActivity(ctx, started.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.service.Pause(ctx, started.SessionID); err != nil {
		t.Fatal(err)
	}
	return advanced.Session
}

func replaceTreeCourse(t *testing.T, fixture treeServiceFixture, pack CoursePack) ReplaceResult {
	t.Helper()
	compiled, err := Compile(pack, chessrules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := fixture.service.catalog.Replace(
		context.Background(), compiled, "/private/tree-v2.ctcourse", "sha-tree-v2",
	)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestOpeningRebaseKeepsMatchingActivityAndJourney(t *testing.T) {
	fixture := newTreeServiceFixture(t)
	paused := pauseTreeLessonAtDecision(t, fixture)
	pack := decodeTreePack(t)
	pack.ContentVersion = "2.0.1"
	pack.Title = "Synthetic Italian Teaching Tree revised"
	activities := pack.Lessons[0].Activities
	pack.Lessons[0].Activities = []LessonActivity{activities[1], activities[0], activities[2]}
	replacement := replaceTreeCourse(t, fixture, pack)

	resumed, err := fixture.service.Resume(context.Background())
	if err != nil || resumed == nil || resumed.Current == nil ||
		resumed.GenerationID != replacement.GenerationID ||
		resumed.Current.ActivityID != "giuoco-c3-decision" {
		t.Fatalf("resumed=%+v err=%v", resumed, err)
	}
	journey, err := fixture.store.Journey(context.Background(), fixture.compiled.Pack.CourseID, DepthReference)
	if err != nil || journey.CurrentActivityID != "giuoco-c3-decision" ||
		journey.ActiveSessionID != paused.SessionID {
		t.Fatalf("journey=%+v err=%v", journey, err)
	}
	stored, err := fixture.store.LoadSession(context.Background(), paused.SessionID)
	if err != nil || stored.ActivityIndex != 0 {
		t.Fatalf("stored=%+v err=%v", stored, err)
	}
}

func TestOpeningRebaseDoesNotSkipRequiredActivityInsertedBeforeCursor(t *testing.T) {
	fixture := newTreeServiceFixture(t)
	paused := pauseTreeLessonAtDecision(t, fixture)
	pack := decodeTreePack(t)
	pack.ContentVersion = "2.0.1"
	inserted := pack.Lessons[0].Activities[0]
	inserted.ActivityID = "inserted-required-concept"
	inserted.Title = "New required concept"
	activities := pack.Lessons[0].Activities
	pack.Lessons[0].Activities = append(
		[]LessonActivity{activities[0], inserted}, activities[1:]...,
	)
	replaceTreeCourse(t, fixture, pack)

	resumed, err := fixture.service.Resume(context.Background())
	if err != nil || resumed == nil || resumed.Current == nil ||
		resumed.Current.ActivityID != "giuoco-c3-decision" {
		t.Fatalf("resumed=%+v err=%v", resumed, err)
	}
	result, err := fixture.service.PlayMove(context.Background(), paused.SessionID, "c2c3")
	if err != nil {
		t.Fatal(err)
	}
	if result.Session.Status != OpeningStatusActive || result.Session.Current == nil ||
		result.Session.Current.ActivityID != inserted.ActivityID || result.Checkpoint != nil {
		t.Fatalf("completion skipped inserted required activity: %+v", result)
	}
}

func TestOpeningRebaseRemovedActivityRestartsAtNearestCompatibleActivity(t *testing.T) {
	fixture := newTreeServiceFixture(t)
	paused := pauseTreeLessonAtDecision(t, fixture)
	pack := decodeTreePack(t)
	pack.ContentVersion = "2.0.1"
	pack.Lessons[0].Activities = append(
		[]LessonActivity{},
		pack.Lessons[0].Activities[0],
		pack.Lessons[0].Activities[2],
	)
	replacement := replaceTreeCourse(t, fixture, pack)

	resumed, err := fixture.service.Resume(context.Background())
	if err != nil || resumed == nil || resumed.Status != OpeningStatusRestartRequired ||
		!strings.Contains(resumed.Notice, "preserved") {
		t.Fatalf("resumed=%+v err=%v", resumed, err)
	}
	stored, err := fixture.store.LoadSession(context.Background(), paused.SessionID)
	if err != nil || stored.State.Restart == nil || stored.State.Restart.ActivityIndex != 0 {
		t.Fatalf("stored=%+v err=%v", stored, err)
	}
	journey, err := fixture.store.Journey(context.Background(), fixture.compiled.Pack.CourseID, DepthReference)
	if err != nil || journey.CurrentActivityID != "giuoco-concept" {
		t.Fatalf("journey=%+v err=%v", journey, err)
	}
	restarted, err := fixture.service.Restart(context.Background(), paused.SessionID)
	if err != nil || restarted.GenerationID != replacement.GenerationID || restarted.Current == nil ||
		restarted.Current.ActivityID != "giuoco-concept" {
		t.Fatalf("restarted=%+v err=%v", restarted, err)
	}
}

func TestOpeningRebaseRemovedLessonRestartsAtFirstVisibleTeachingNode(t *testing.T) {
	fixture := newTreeServiceFixture(t)
	paused := pauseTreeLessonAtDecision(t, fixture)
	pack := decodeTreePack(t)
	pack.ContentVersion = "2.0.1"
	replacementLesson := pack.Lessons[0]
	replacementLesson.LessonID = "replacement-plan"
	replacementLesson.Title = "Replacement plan"
	pack.Lessons = []Lesson{replacementLesson}
	pack.LessonEdges = []LessonEdge{}
	replacement := replaceTreeCourse(t, fixture, pack)

	resumed, err := fixture.service.Resume(context.Background())
	if err != nil || resumed == nil || resumed.Status != OpeningStatusRestartRequired ||
		!strings.Contains(resumed.Notice, "no longer available") {
		t.Fatalf("resumed=%+v err=%v", resumed, err)
	}
	restarted, err := fixture.service.Restart(context.Background(), paused.SessionID)
	if err != nil || restarted.GenerationID != replacement.GenerationID ||
		restarted.LessonID != "replacement-plan" || restarted.Current == nil ||
		restarted.Current.ActivityID != "giuoco-concept" {
		t.Fatalf("restarted=%+v err=%v", restarted, err)
	}
	journey, err := fixture.store.Journey(context.Background(), fixture.compiled.Pack.CourseID, DepthReference)
	if err != nil || journey.CurrentLessonID != "replacement-plan" ||
		journey.CurrentActivityID != "giuoco-concept" {
		t.Fatalf("journey=%+v err=%v", journey, err)
	}
}

func replaceMiniCourse(t *testing.T, fixture openingServiceFixture, pack CoursePack) ReplaceResult {
	t.Helper()
	compiled, err := Compile(pack, chessrules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := fixture.catalog.Replace(
		context.Background(), compiled, "/private/v2.ctcourse", "sha-v2",
	)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestOpeningLessonSessionRebasesUnchangedPromptInPlace(t *testing.T) {
	ctx := context.Background()
	fixture := newOpeningServiceFixture(t)
	paused := pauseMiniLessonAtRecall(t, fixture)
	pack := decodeMiniPack(t)
	pack.ContentVersion = "2.0.0"
	pack.Title = "Synthetic Italian revised"
	v2 := replaceMiniCourse(t, fixture, pack)

	resumed, err := fixture.service.Resume(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if resumed == nil || resumed.GenerationID != v2.GenerationID || resumed.Status != OpeningStatusActive ||
		resumed.Current == nil || resumed.Current.ActivityID != "recall-c3-step" {
		t.Fatalf("resumed = %+v", resumed)
	}
	stored, err := fixture.store.LoadSession(ctx, paused.SessionID)
	if err != nil || stored.GenerationID != v2.GenerationID {
		t.Fatalf("stored rebase = %+v err=%v", stored, err)
	}
}

func TestOpeningLessonChangedPromptRequiresCheckpointRestart(t *testing.T) {
	ctx := context.Background()
	fixture := newOpeningServiceFixture(t)
	paused := pauseMiniLessonAtRecall(t, fixture)
	pack := decodeMiniPack(t)
	pack.ContentVersion = "2.0.0"
	pack.Prompts[0].AcceptedAlternativeMoveIDs = []string{}
	v2 := replaceMiniCourse(t, fixture, pack)

	resumed, err := fixture.service.Resume(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if resumed == nil || resumed.Status != OpeningStatusRestartRequired || resumed.Current != nil ||
		!strings.Contains(resumed.Notice, "course was updated") {
		t.Fatalf("restart-required view = %+v", resumed)
	}
	stored, err := fixture.store.LoadSession(ctx, paused.SessionID)
	if err != nil || stored.GenerationID == v2.GenerationID || stored.State.Restart == nil ||
		stored.State.Restart.ActivityIndex != 1 {
		t.Fatalf("restart checkpoint = %+v err=%v", stored, err)
	}
	var attemptsBefore int
	if err := fixture.userDB.QueryRow(`SELECT COUNT(*) FROM opening_attempts`).Scan(&attemptsBefore); err != nil {
		t.Fatal(err)
	}
	restarted, err := fixture.service.Restart(ctx, paused.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if restarted.GenerationID != v2.GenerationID || restarted.Status != OpeningStatusActive ||
		restarted.Current == nil || restarted.Current.ActivityID != "watch-setup" {
		t.Fatalf("restarted = %+v", restarted)
	}
	var attemptsAfter int
	if err := fixture.userDB.QueryRow(`SELECT COUNT(*) FROM opening_attempts`).Scan(&attemptsAfter); err != nil {
		t.Fatal(err)
	}
	if attemptsAfter != attemptsBefore {
		t.Fatalf("attempt history changed from %d to %d", attemptsBefore, attemptsAfter)
	}
}

func TestOpeningRemovedLessonRestartsAtFirstVisibleLesson(t *testing.T) {
	ctx := context.Background()
	fixture := newOpeningServiceFixture(t)
	paused := pauseMiniLessonAtRecall(t, fixture)
	pack := decodeMiniPack(t)
	pack.ContentVersion = "2.0.0"
	fallback := pack.Lessons[0]
	fallback.LessonID = "replacement-lesson"
	fallback.Title = "Replacement lesson"
	pack.Lessons = []Lesson{fallback}
	v2 := replaceMiniCourse(t, fixture, pack)

	resumed, err := fixture.service.Resume(ctx)
	if err != nil || resumed == nil || resumed.Status != OpeningStatusRestartRequired {
		t.Fatalf("resume = %+v err=%v", resumed, err)
	}
	restarted, err := fixture.service.Restart(ctx, paused.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if restarted.GenerationID != v2.GenerationID || restarted.LessonID != "replacement-lesson" ||
		restarted.Current == nil || restarted.Current.ActivityID != "explain-plan" {
		t.Fatalf("replacement restart = %+v", restarted)
	}
}

func TestOpeningResumeMissingGenerationRequestsPrivateReimport(t *testing.T) {
	ctx := context.Background()
	fixture := newOpeningServiceFixture(t)
	pauseMiniLessonAtRecall(t, fixture)
	pack := decodeMiniPack(t)
	pack.ContentVersion = "2.0.0"
	replaceMiniCourse(t, fixture, pack)
	if _, err := fixture.catalog.CleanupBatch(ctx, map[string]struct{}{}, 100); err != nil {
		t.Fatal(err)
	}

	resumed, err := fixture.service.Resume(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if resumed == nil || resumed.Status != OpeningStatusRestartRequired ||
		!strings.Contains(strings.ToLower(resumed.Notice), "reimport") {
		t.Fatalf("missing-generation resume = %+v", resumed)
	}
}

func TestOpeningReviewRebaseKeepsOnlyUnchangedQueuedPrompts(t *testing.T) {
	ctx := context.Background()
	fixture := newOpeningServiceFixture(t)
	due := fixture.now.Add(-1).UnixMilli()
	for _, promptID := range []string{"recall-c3", "recall-d3"} {
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
	pack := decodeMiniPack(t)
	pack.ContentVersion = "2.0.0"
	pack.Prompts[0].AcceptedAlternativeMoveIDs = []string{}
	v2 := replaceMiniCourse(t, fixture, pack)

	resumed, err := fixture.service.Resume(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if resumed == nil || resumed.GenerationID != v2.GenerationID || resumed.Current == nil ||
		resumed.Current.PositionID != "after-nf6" {
		t.Fatalf("rebased review = %+v", resumed)
	}
	stored, err := fixture.store.LoadSession(ctx, review.SessionID)
	if err != nil || stored.State.Review == nil ||
		!slices.Equal(stored.State.Review.PromptIDs, []string{"recall-d3"}) || stored.State.Review.Index != 0 {
		t.Fatalf("rebased review queue = %+v err=%v", stored.State, err)
	}
	var status string
	if err := fixture.userDB.QueryRow(
		`SELECT status FROM opening_review_state WHERE course_id = ? AND prompt_id = 'recall-c3'`,
		fixture.compiled.Pack.CourseID,
	).Scan(&status); err != nil || status != "archived" {
		t.Fatalf("changed review status = %q err=%v", status, err)
	}
}

func TestOpeningReviewRebaseCompletesWhenNoQueuedPromptSurvives(t *testing.T) {
	ctx := context.Background()
	fixture := newOpeningServiceFixture(t)
	due := fixture.now.Add(-1).UnixMilli()
	for _, promptID := range []string{"recall-c3", "recall-d3"} {
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
	if _, err := fixture.service.StartReview(ctx, fixture.compiled.Pack.CourseID); err != nil {
		t.Fatal(err)
	}
	pack := decodeMiniPack(t)
	pack.ContentVersion = "2.0.0"
	pack.Prompts[0].AcceptedAlternativeMoveIDs = []string{}
	pack.Prompts = pack.Prompts[:1]
	replaceMiniCourse(t, fixture, pack)

	resumed, err := fixture.service.Resume(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if resumed == nil || resumed.Status != OpeningStatusCompleted || resumed.Summary == nil ||
		!strings.Contains(resumed.Notice, "no queued review positions remain") {
		t.Fatalf("completed rebased review = %+v", resumed)
	}
}

func TestOpeningReviewWorkflowArchivesChangedPromptWithoutMovingMastery(t *testing.T) {
	ctx := context.Background()
	fixture := newOpeningServiceFixture(t)
	oldPrompt := fixture.compiled.Prompts["recall-c3"]
	if _, err := fixture.userDB.Exec(
		`INSERT INTO opening_review_state(
		 course_id, prompt_id, semantic_fingerprint, due_at, interval_index,
		 successful_reviews, last_outcome, status
		) VALUES (?, 'recall-c3', ?, ?, 3, 4, 'clean', 'active')`,
		fixture.compiled.Pack.CourseID, oldPrompt.SemanticFingerprint, fixture.now.UnixMilli(),
	); err != nil {
		t.Fatal(err)
	}
	pack := decodeMiniPack(t)
	pack.ContentVersion = "2.0.0"
	pack.Prompts[0].AcceptedAlternativeMoveIDs = []string{}
	replaceMiniCourse(t, fixture, pack)

	if _, err := fixture.service.Home(ctx); err != nil {
		t.Fatal(err)
	}
	readMastery := func() (string, int, string) {
		t.Helper()
		var fingerprint, status string
		var successes int
		if err := fixture.userDB.QueryRow(
			`SELECT semantic_fingerprint, successful_reviews, status
			 FROM opening_review_state WHERE course_id = ? AND prompt_id = 'recall-c3'`,
			fixture.compiled.Pack.CourseID,
		).Scan(&fingerprint, &successes, &status); err != nil {
			t.Fatal(err)
		}
		return fingerprint, successes, status
	}
	fingerprint, successes, status := readMastery()
	if fingerprint != oldPrompt.SemanticFingerprint || successes != 4 || status != "active" {
		t.Fatalf("home mutated mastery = fingerprint %q successes %d status %q", fingerprint, successes, status)
	}
	if _, err := fixture.service.StartReview(ctx, fixture.compiled.Pack.CourseID); err == nil ||
		!strings.Contains(err.Error(), "no opening reviews are due") {
		t.Fatalf("StartReview() error = %v", err)
	}
	fingerprint, successes, status = readMastery()
	if fingerprint != oldPrompt.SemanticFingerprint || successes != 4 || status != "archived" {
		t.Fatalf("archived mastery = fingerprint %q successes %d status %q", fingerprint, successes, status)
	}
}

func TestSessionAwareMaintenanceProtectsReferencedOldGeneration(t *testing.T) {
	ctx := context.Background()
	fixture := newOpeningServiceFixture(t)
	paused := pauseMiniLessonAtRecall(t, fixture)
	pack := decodeMiniPack(t)
	pack.ContentVersion = "2.0.0"
	v2 := replaceMiniCourse(t, fixture, pack)
	maintenance := SessionAwareMaintenance{Catalog: fixture.catalog, Store: fixture.store}

	if more, err := maintenance.CleanupBatch(ctx, 100); err != nil || more {
		t.Fatalf("protected cleanup more=%v err=%v", more, err)
	}
	if _, err := fixture.catalog.LoadGeneration(ctx, fixture.result.GenerationID); err != nil {
		t.Fatalf("protected V1 was deleted: %v", err)
	}
	if err := fixture.store.SetSessionStatus(ctx, paused.SessionID, OpeningStatusCompleted, fixture.now); err != nil {
		t.Fatal(err)
	}
	if more, err := maintenance.CleanupBatch(ctx, 100); err != nil || more {
		t.Fatalf("unprotected cleanup more=%v err=%v", more, err)
	}
	if _, err := fixture.catalog.LoadGeneration(ctx, fixture.result.GenerationID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unprotected V1 error = %v, want sql.ErrNoRows", err)
	}
	active, err := fixture.catalog.ActiveGenerationID(ctx, fixture.compiled.Pack.CourseID)
	if err != nil || active != v2.GenerationID {
		t.Fatalf("active generation = %q err=%v", active, err)
	}
}
