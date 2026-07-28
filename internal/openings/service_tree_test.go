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

func TestActiveOpeningLessonIncludesConciseLocalCourseMap(t *testing.T) {
	fixture := newTreeServiceFixture(t)
	started, err := fixture.service.StartLesson(
		context.Background(), fixture.compiled.Pack.CourseID, "giuoco-plan",
	)
	if err != nil {
		t.Fatal(err)
	}
	if started.CourseMap == nil {
		t.Fatal("active lesson course map is missing")
	}
	courseMap := started.CourseMap
	if courseMap.CompletedIdeas != 0 || courseMap.TotalIdeas != 4 {
		t.Fatalf("course map progress = %d/%d, want 0/4", courseMap.CompletedIdeas, courseMap.TotalIdeas)
	}
	if len(courseMap.Entries) != 2 {
		t.Fatalf("course map entries = %#v", courseMap.Entries)
	}
	current := courseMap.Entries[0]
	if current.LessonID != "giuoco-plan" || current.ParentLessonID != "" ||
		current.MoveLabel != "3...Bc5" || current.Title != "Prepare the center" ||
		current.State != CourseMapCurrent || !current.Selectable {
		t.Fatalf("current course map entry = %+v", current)
	}
	next := courseMap.Entries[1]
	if next.LessonID != "two-knights-plan" || next.ParentLessonID != "giuoco-plan" ||
		next.MoveLabel != "3...Nf6" || next.Title != "Choose the quiet setup" ||
		next.State != CourseMapNext || !next.Selectable {
		t.Fatalf("next course map entry = %+v", next)
	}
}

func TestLocalCourseMapIncludesVisibleSiblingAndOmitsHiddenSibling(t *testing.T) {
	fixture := newTreeServiceFixture(t)
	fixture.completeLesson(t, "giuoco-plan")
	fixture.saveJourney(t, CourseJourney{
		CurrentLessonID: "two-knights-plan",
		PathLessonIDs:   []string{"giuoco-plan", "two-knights-plan"},
	})

	course := fixture.compiled
	oneIdea := func(activityID, positionID string) []LessonActivity {
		return []LessonActivity{{
			ActivityID: activityID, Kind: ActivityConcept, Title: activityID,
			Instruction: activityID, Required: true, PositionID: positionID,
		}}
	}
	additionalLessons := []Lesson{
		{
			LessonID: "quiet-followup", ChapterID: "two-knights", Title: "Build the quiet position",
			MinimumDepth: DepthStandard, StartPositionID: "after-d3",
			Activities: oneIdea("quiet-followup-concept", "after-d3"),
		},
		{
			LessonID: "giuoco-alternative", ChapterID: "giuoco", Title: "Try another central plan",
			MinimumDepth: DepthStandard, StartPositionID: "after-c3",
			Activities: oneIdea("giuoco-alternative-concept", "after-c3"),
		},
		{
			LessonID: "hidden-deep-line", ChapterID: "giuoco", Title: "A deeper alternative",
			MinimumDepth: DepthReference, StartPositionID: "after-b4",
			Activities: oneIdea("hidden-deep-line-concept", "after-b4"),
		},
	}
	for _, lesson := range additionalLessons {
		course.Lessons[lesson.LessonID] = lesson
		course.Pack.Lessons = append(course.Pack.Lessons, lesson)
	}
	followupEdge := LessonEdge{
		EdgeID: "quiet-followup", FromLessonID: "two-knights-plan", ToLessonID: "quiet-followup",
		Ordinal: 1, Kind: EdgeContinuation, MinimumDepth: DepthStandard,
	}
	alternativeEdge := LessonEdge{
		EdgeID: "giuoco-alternative", FromLessonID: "giuoco-plan", ToLessonID: "giuoco-alternative",
		Ordinal: 2, Kind: EdgeAlternative, MinimumDepth: DepthStandard,
	}
	hiddenEdge := LessonEdge{
		EdgeID: "hidden-deep-line", FromLessonID: "giuoco-plan", ToLessonID: "hidden-deep-line",
		Ordinal: 3, Kind: EdgeAlternative, MinimumDepth: DepthReference,
	}
	course.LessonChildren["two-knights-plan"] = []LessonEdge{followupEdge}
	course.LessonChildren["giuoco-plan"] = append(
		course.LessonChildren["giuoco-plan"], alternativeEdge, hiddenEdge,
	)
	course.LessonParent["quiet-followup"] = followupEdge
	course.LessonParent["giuoco-alternative"] = alternativeEdge
	course.LessonParent["hidden-deep-line"] = hiddenEdge
	course.Pack.LessonEdges = append(
		course.Pack.LessonEdges, followupEdge, alternativeEdge, hiddenEdge,
	)

	courseMap, err := fixture.service.localCourseMap(
		context.Background(), course, DepthStandard, "two-knights-plan",
	)
	if err != nil {
		t.Fatal(err)
	}
	if courseMap.CompletedIdeas != 3 || courseMap.TotalIdeas != 6 {
		t.Fatalf("course map progress = %d/%d, want 3/6", courseMap.CompletedIdeas, courseMap.TotalIdeas)
	}
	entries := map[string]OpeningCourseMapEntryView{}
	for _, entry := range courseMap.Entries {
		entries[entry.LessonID] = entry
	}
	if _, exists := entries["hidden-deep-line"]; exists {
		t.Fatalf("hidden sibling is present: %#v", courseMap.Entries)
	}
	alternative, exists := entries["giuoco-alternative"]
	if !exists || alternative.State != CourseMapAvailable || alternative.ParentLessonID != "giuoco-plan" {
		t.Fatalf("visible sibling = %+v, exists=%t", alternative, exists)
	}
	followup, exists := entries["quiet-followup"]
	if !exists || followup.State != CourseMapNext || followup.ParentLessonID != "two-knights-plan" {
		t.Fatalf("next child = %+v, exists=%t", followup, exists)
	}
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
	_, err := fixture.store.CreateSession(ctx, SessionSeed{
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
		CurrentLessonID: "two-knights-plan",
		PathLessonIDs:   []string{"giuoco-plan", "two-knights-plan"},
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
	_, err := fixture.store.CreateSession(context.Background(), SessionSeed{
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
		CurrentLessonID: "two-knights-plan",
		PathLessonIDs:   []string{"giuoco-plan", "two-knights-plan"},
	})
	if err := fixture.service.SetDepth(context.Background(), fixture.compiled.Pack.CourseID, DepthQuick); err != nil {
		t.Fatal(err)
	}
	journey, err := fixture.store.Journey(context.Background(), fixture.compiled.Pack.CourseID)
	if err != nil {
		t.Fatal(err)
	}
	depth, depthErr := fixture.store.Depth(context.Background(), fixture.compiled.Pack.CourseID, DepthReference)
	if depthErr != nil || depth != DepthQuick || journey.CurrentLessonID != "two-knights-plan" || len(journey.PathLessonIDs) != 2 {
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
