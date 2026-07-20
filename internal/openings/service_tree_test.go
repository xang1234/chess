package openings

import (
	"context"
	"testing"
	"time"

	"chess-trainer/internal/chessrules"
)

type treeServiceFixture struct {
	service  *Service
	store    *UserStore
	compiled CompiledCourse
	result   ReplaceResult
	now      time.Time
}

func newTreeServiceFixture(t *testing.T) treeServiceFixture {
	t.Helper()
	ctx := context.Background()
	compiled := compileTreeCourse(t)
	catalog := NewSQLiteCatalog(openCourseCatalogTestDB(t))
	result, err := catalog.Replace(ctx, compiled, "/private/tree.ctcourse", "sha-tree")
	if err != nil {
		t.Fatal(err)
	}
	store := NewUserStore(openOpeningUserTestDB(t))
	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	service := NewService(catalog, store, chessrules.Rules{}, "")
	service.now = func() time.Time { return now }
	return treeServiceFixture{service: service, store: store, compiled: compiled, result: result, now: now}
}

func (fixture treeServiceFixture) saveJourney(t *testing.T, journey CourseJourney) {
	t.Helper()
	journey.CourseID = fixture.compiled.Pack.CourseID
	if journey.Depth == "" {
		journey.Depth = fixture.compiled.Pack.DefaultDepth
	}
	journey.CreatedAt = fixture.now
	journey.UpdatedAt = fixture.now
	if journey.PathLessonIDs == nil {
		journey.PathLessonIDs = []string{}
	}
	if err := fixture.store.SaveJourney(context.Background(), journey); err != nil {
		t.Fatal(err)
	}
}

func (fixture treeServiceFixture) completeLesson(t *testing.T, lessonID string) {
	t.Helper()
	lesson := fixture.compiled.Lessons[lessonID]
	required := RequiredActivityIDs(lesson)
	for _, activityID := range required {
		if err := fixture.store.RecordActivityProgress(context.Background(), ActivityProgressUpdate{
			CourseID: fixture.compiled.Pack.CourseID, LessonID: lessonID,
			CompletedActivityID: activityID, RequiredActivityIDs: required, Now: fixture.now,
		}); err != nil {
			t.Fatal(err)
		}
	}
}

func treeNode(t *testing.T, course OpeningCourseSummary, lessonID string) OpeningTeachingNodeView {
	t.Helper()
	for _, node := range course.Tree.Nodes {
		if node.LessonID == lessonID {
			return node
		}
	}
	t.Fatalf("tree node %q is missing", lessonID)
	return OpeningTeachingNodeView{}
}

func TestOpeningHomeReturnsTeachingTreeAndRecommendation(t *testing.T) {
	fixture := newTreeServiceFixture(t)
	home, err := fixture.service.Home(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(home.Courses) != 1 {
		t.Fatalf("courses = %#v", home.Courses)
	}
	course := home.Courses[0]
	if course.Tree.RootLessonID != "giuoco-plan" || len(course.Tree.Nodes) != 2 || len(course.Tree.Edges) != 1 {
		t.Fatalf("tree = %+v", course.Tree)
	}
	root := treeNode(t, course, "giuoco-plan")
	if course.RecommendedLessonID != "giuoco-plan" || course.RecommendedLessonTitle != root.Title || !root.Recommended {
		t.Fatalf("course = %+v root = %+v", course, root)
	}
	if root.Progress != NodeAvailable || !root.Visible || root.RequiredActivities != 3 {
		t.Fatalf("root = %+v", root)
	}
}

func TestRecommendationUsesExactVisibleResumableLesson(t *testing.T) {
	fixture := newTreeServiceFixture(t)
	ctx := context.Background()
	session, err := fixture.store.CreateSession(ctx, SessionSeed{
		CourseID: fixture.compiled.Pack.CourseID, GenerationID: fixture.result.GenerationID,
		LessonID: "two-knights-plan", Mode: OpeningModeLesson, Depth: DepthReference,
		State: SessionState{Position: PositionState{
			PositionID: "after-nf6", CurrentFEN: fixture.compiled.Positions["after-nf6"].FEN,
			PlayedMoveIDs: []string{},
		}},
	}, fixture.now)
	if err != nil {
		t.Fatal(err)
	}
	fixture.saveJourney(t, CourseJourney{
		CurrentLessonID: "two-knights-plan", CurrentActivityID: "two-knights-d3-decision",
		PathLessonIDs: []string{"giuoco-plan", "two-knights-plan"}, ActiveSessionID: session.ID,
	})

	home, err := fixture.service.Home(ctx)
	if err != nil {
		t.Fatal(err)
	}
	course := home.Courses[0]
	node := treeNode(t, course, "two-knights-plan")
	if course.RecommendedLessonID != "two-knights-plan" || !course.HasResumable ||
		course.CurrentLessonID != "two-knights-plan" || course.CurrentActivityID != "two-knights-d3-decision" ||
		node.Progress != NodeInProgress || !node.Recommended {
		t.Fatalf("course = %+v node = %+v", course, node)
	}
	if len(course.CurrentPath) != 2 || course.CurrentPath[1].LessonID != "two-knights-plan" {
		t.Fatalf("current path = %#v", course.CurrentPath)
	}
}

func TestOpeningHomeProjectsLegacyResumableSessionIntoCurrentJourney(t *testing.T) {
	fixture := newTreeServiceFixture(t)
	ctx := context.Background()
	if _, err := fixture.store.CreateSession(ctx, SessionSeed{
		CourseID: fixture.compiled.Pack.CourseID, GenerationID: fixture.result.GenerationID,
		LessonID: "two-knights-plan", Mode: OpeningModeLesson, Depth: DepthReference,
		ActivityIndex: 0,
		State: SessionState{Position: PositionState{
			PositionID: "after-nf6", CurrentFEN: fixture.compiled.Positions["after-nf6"].FEN,
			PlayedMoveIDs: []string{},
		}},
	}, fixture.now); err != nil {
		t.Fatal(err)
	}

	home, err := fixture.service.Home(ctx)
	if err != nil {
		t.Fatal(err)
	}
	course := home.Courses[0]
	if course.CurrentLessonID != "two-knights-plan" ||
		course.CurrentActivityID != "two-knights-d3-decision" {
		t.Fatalf("current journey = lesson %q activity %q", course.CurrentLessonID, course.CurrentActivityID)
	}
	if len(course.CurrentPath) != 2 || course.CurrentPath[0].LessonID != "giuoco-plan" ||
		course.CurrentPath[1].LessonID != "two-knights-plan" {
		t.Fatalf("current path = %#v", course.CurrentPath)
	}
}

func TestRecommendationContinuesFromDeepestCompletedPathNode(t *testing.T) {
	fixture := newTreeServiceFixture(t)
	fixture.completeLesson(t, "giuoco-plan")
	fixture.saveJourney(t, CourseJourney{PathLessonIDs: []string{"giuoco-plan"}})

	home, err := fixture.service.Home(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	course := home.Courses[0]
	if course.RecommendedLessonID != "two-knights-plan" ||
		treeNode(t, course, "giuoco-plan").Progress != NodeCompleted ||
		!treeNode(t, course, "two-knights-plan").Recommended {
		t.Fatalf("course = %+v", course)
	}
}

func TestRecommendationPrefersContinuationBeforeAuthoredAlternatives(t *testing.T) {
	lessons := map[string]Lesson{
		"root":         {LessonID: "root", MinimumDepth: DepthQuick},
		"alternative":  {LessonID: "alternative", MinimumDepth: DepthQuick},
		"continuation": {LessonID: "continuation", MinimumDepth: DepthQuick},
		"reference":    {LessonID: "reference", MinimumDepth: DepthQuick},
	}
	course := CompiledCourse{
		Pack: CoursePack{CourseID: "course", Lessons: []Lesson{
			lessons["root"], lessons["alternative"], lessons["continuation"], lessons["reference"],
		}},
		Lessons: lessons, RootLessonID: "root",
		LessonChildren: map[string][]LessonEdge{
			"root": {
				{EdgeID: "alternative", FromLessonID: "root", ToLessonID: "alternative", Ordinal: 1, Kind: EdgeAlternative},
				{EdgeID: "continuation", FromLessonID: "root", ToLessonID: "continuation", Ordinal: 2, Kind: EdgeContinuation},
				{EdgeID: "reference", FromLessonID: "root", ToLessonID: "reference", Ordinal: 3, Kind: EdgeReference},
			},
		},
	}
	progress := map[string]LessonProgress{"root": {Completed: true}}
	if got := recommendTeachingLesson(course, DepthQuick, CourseJourney{}, nil, progress); got != "continuation" {
		t.Fatalf("recommendation = %q, want continuation", got)
	}
	progress["continuation"] = LessonProgress{Completed: true}
	if got := recommendTeachingLesson(course, DepthQuick, CourseJourney{}, nil, progress); got != "alternative" {
		t.Fatalf("recommendation = %q, want authored alternative", got)
	}
	progress["alternative"] = LessonProgress{Completed: true}
	if got := recommendTeachingLesson(course, DepthQuick, CourseJourney{}, nil, progress); got != "reference" {
		t.Fatalf("recommendation = %q, want authored reference", got)
	}
}

func TestRecommendationPreservesHiddenTreeContextAtQuickDepth(t *testing.T) {
	fixture := newTreeServiceFixture(t)
	if err := fixture.service.SetDepth(context.Background(), fixture.compiled.Pack.CourseID, DepthQuick); err != nil {
		t.Fatal(err)
	}
	home, err := fixture.service.Home(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	course := home.Courses[0]
	root := treeNode(t, course, "giuoco-plan")
	hidden := treeNode(t, course, "two-knights-plan")
	if len(course.Tree.Nodes) != 2 || !root.Visible || hidden.Visible || hidden.Recommended ||
		course.TotalLessons != 1 || course.RecommendedLessonID != "giuoco-plan" {
		t.Fatalf("course = %+v root = %+v hidden = %+v", course, root, hidden)
	}
}

func TestRecommendationIsEmptyWhenEveryVisibleNodeIsComplete(t *testing.T) {
	fixture := newTreeServiceFixture(t)
	fixture.completeLesson(t, "giuoco-plan")
	fixture.completeLesson(t, "two-knights-plan")

	home, err := fixture.service.Home(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	course := home.Courses[0]
	if course.CompletedLessons != 2 || course.TotalLessons != 2 ||
		course.RecommendedLessonID != "" || course.RecommendedLessonTitle != "" {
		t.Fatalf("course = %+v", course)
	}
	for _, node := range course.Tree.Nodes {
		if node.Visible && (node.Progress != NodeCompleted || node.Recommended) {
			t.Fatalf("node = %+v", node)
		}
	}
}

func TestOpeningHomeMapsDueReviewToLessonNode(t *testing.T) {
	fixture := newTreeServiceFixture(t)
	prompt := fixture.compiled.Prompts["recall-d3"]
	if _, err := fixture.store.db.Exec(
		`INSERT INTO opening_review_state(
		 course_id, prompt_id, semantic_fingerprint, due_at, interval_index,
		 successful_reviews, last_outcome, status
		) VALUES (?, ?, ?, ?, 0, 1, 'clean', 'active')`,
		fixture.compiled.Pack.CourseID, prompt.PromptID, prompt.SemanticFingerprint,
		fixture.now.Add(-time.Minute).UnixMilli(),
	); err != nil {
		t.Fatal(err)
	}

	home, err := fixture.service.Home(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	course := home.Courses[0]
	if course.DueReviews != 1 || !treeNode(t, course, "two-knights-plan").ReviewDue ||
		treeNode(t, course, "giuoco-plan").ReviewDue {
		t.Fatalf("course = %+v", course)
	}
}

func TestPausedReviewStaysSeparateFromContinueLearning(t *testing.T) {
	fixture := newTreeServiceFixture(t)
	prompt := fixture.compiled.Prompts["recall-d3"]
	if _, err := fixture.store.db.Exec(
		`INSERT INTO opening_review_state(
		 course_id, prompt_id, semantic_fingerprint, due_at, interval_index,
		 successful_reviews, last_outcome, status
		) VALUES (?, ?, ?, ?, 0, 1, 'clean', 'active')`,
		fixture.compiled.Pack.CourseID, prompt.PromptID, prompt.SemanticFingerprint,
		fixture.now.Add(-time.Minute).UnixMilli(),
	); err != nil {
		t.Fatal(err)
	}
	started, err := fixture.service.StartReview(context.Background(), fixture.compiled.Pack.CourseID)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.service.Pause(context.Background(), started.SessionID); err != nil {
		t.Fatal(err)
	}

	home, err := fixture.service.Home(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	course := home.Courses[0]
	if course.HasResumable || !course.HasResumableReview ||
		course.RecommendedLessonID != "giuoco-plan" {
		t.Fatalf("paused review leaked into lesson continuation: %+v", course)
	}
	resumed, err := fixture.service.StartReview(context.Background(), fixture.compiled.Pack.CourseID)
	if err != nil || resumed.SessionID != started.SessionID || resumed.Mode != OpeningModeReview ||
		resumed.Status != OpeningStatusActive {
		t.Fatalf("resumed review=%+v err=%v", resumed, err)
	}
}

func TestSetDepthPersistsJourneyAndRecommendsAroundHiddenCurrentLesson(t *testing.T) {
	fixture := newTreeServiceFixture(t)
	session, err := fixture.store.CreateSession(context.Background(), SessionSeed{
		CourseID: fixture.compiled.Pack.CourseID, GenerationID: fixture.result.GenerationID,
		LessonID: "two-knights-plan", Mode: OpeningModeLesson, Depth: DepthReference,
		State: SessionState{Position: PositionState{
			PositionID: "after-nf6", CurrentFEN: fixture.compiled.Positions["after-nf6"].FEN,
			PlayedMoveIDs: []string{},
		}},
	}, fixture.now)
	if err != nil {
		t.Fatal(err)
	}
	fixture.saveJourney(t, CourseJourney{
		CurrentLessonID: "two-knights-plan", CurrentActivityID: "two-knights-d3-decision",
		PathLessonIDs: []string{"giuoco-plan", "two-knights-plan"}, ActiveSessionID: session.ID,
	})
	if err := fixture.service.SetDepth(context.Background(), fixture.compiled.Pack.CourseID, DepthQuick); err != nil {
		t.Fatal(err)
	}
	journey, err := fixture.store.Journey(context.Background(), fixture.compiled.Pack.CourseID, DepthReference)
	if err != nil {
		t.Fatal(err)
	}
	if journey.Depth != DepthQuick || journey.CurrentLessonID != "two-knights-plan" || len(journey.PathLessonIDs) != 2 {
		t.Fatalf("journey = %+v", journey)
	}
	home, err := fixture.service.Home(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	course := home.Courses[0]
	if course.RecommendedLessonID != "giuoco-plan" || course.CurrentLessonID != "two-knights-plan" ||
		course.HasResumable || treeNode(t, course, "two-knights-plan").Visible {
		t.Fatalf("course = %+v", course)
	}
}
