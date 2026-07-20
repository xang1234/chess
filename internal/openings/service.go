package openings

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const reviewLessonID = "review"

type Service struct {
	catalog *SQLiteCatalog
	store   *UserStore
	rules   RulesPort
	now     func() time.Time
	notice  string
}

func NewService(
	catalog *SQLiteCatalog,
	store *UserStore,
	rules RulesPort,
	notice string,
) *Service {
	return &Service{
		catalog: catalog,
		store:   store,
		rules:   rules,
		now:     time.Now,
		notice:  strings.TrimSpace(notice),
	}
}

func (s *Service) Home(ctx context.Context) (OpeningHomeView, error) {
	if err := s.validate(); err != nil {
		return OpeningHomeView{}, err
	}
	summaries, err := s.catalog.ListActive(ctx)
	if err != nil {
		return OpeningHomeView{}, err
	}
	resumable, err := s.store.ResumableSession(ctx)
	if err != nil {
		return OpeningHomeView{}, fmt.Errorf("load resumable opening session: %w", err)
	}
	home := OpeningHomeView{Notice: s.notice, Courses: make([]OpeningCourseSummary, 0, len(summaries))}
	for _, summary := range summaries {
		course, err := s.catalog.LoadGeneration(ctx, summary.GenerationID)
		if err != nil {
			return OpeningHomeView{}, fmt.Errorf("load opening course %q: %w", summary.CourseID, err)
		}
		depth, err := s.store.Depth(ctx, summary.CourseID, summary.DefaultDepth)
		if err != nil {
			return OpeningHomeView{}, fmt.Errorf("load opening depth for %q: %w", summary.CourseID, err)
		}
		view, err := s.courseSummary(ctx, course, depth, resumable)
		if err != nil {
			return OpeningHomeView{}, err
		}
		home.Courses = append(home.Courses, view)
	}
	return home, nil
}

func (s *Service) courseSummary(
	ctx context.Context,
	course CompiledCourse,
	fallbackDepth Depth,
	resumable *StoredSession,
) (OpeningCourseSummary, error) {
	journey, err := s.store.Journey(ctx, course.Pack.CourseID, fallbackDepth)
	if err != nil {
		return OpeningCourseSummary{}, fmt.Errorf("load opening journey for %q: %w", course.Pack.CourseID, err)
	}
	depth := journey.Depth
	projection, err := s.projectTeachingTree(ctx, course, depth, journey, resumable)
	if err != nil {
		return OpeningCourseSummary{}, err
	}
	currentLessonID := validJourneyLessonID(course, journey.CurrentLessonID)
	currentActivityID := journey.CurrentActivityID
	if currentLessonID == "" {
		currentActivityID = ""
	}
	view := OpeningCourseSummary{
		CourseID: course.Pack.CourseID, Title: course.Pack.Title,
		Perspective: course.Pack.Perspective, Depth: depth,
		RootPositionID:         course.Pack.RootPositionID,
		CompletedLessons:       projection.completedLessons,
		TotalLessons:           projection.totalLessons,
		DueReviews:             projection.dueReviews,
		CurrentLessonID:        currentLessonID,
		CurrentActivityID:      currentActivityID,
		CurrentPath:            projection.currentPath,
		RecommendedLessonID:    projection.recommendedLessonID,
		RecommendedLessonTitle: projection.recommendedLessonTitle,
		NextLessonID:           projection.recommendedLessonID,
		NextLessonTitle:        projection.recommendedLessonTitle,
		Tree:                   projection.tree,
		Chapters:               []OpeningChapterSummary{},
	}
	view.HasResumable = hasVisibleResumable(course, depth, resumable)
	selectedRank, _ := depthRank(depth)
	for _, chapter := range course.Pack.Chapters {
		chapterRank, ok := depthRank(chapter.MinimumDepth)
		if !ok || chapterRank > selectedRank {
			continue
		}
		chapterView := OpeningChapterSummary{
			ChapterID: chapter.ChapterID,
			Title:     chapter.Title,
			Lessons:   []OpeningLessonSummary{},
		}
		for _, lesson := range course.Pack.Lessons {
			if lesson.ChapterID != chapter.ChapterID || !visibleAtDepth(lesson.MinimumDepth, depth) {
				continue
			}
			progress := projection.progressByLesson[lesson.LessonID]
			lessonView := OpeningLessonSummary{
				LessonID: lesson.LessonID, Title: lesson.Title,
				CompletedSteps: progress.CompletedActivities,
				TotalSteps:     progress.TotalActivities,
				Completed:      progress.Completed,
			}
			chapterView.Lessons = append(chapterView.Lessons, lessonView)
		}
		view.Chapters = append(view.Chapters, chapterView)
	}
	return view, nil
}

func (s *Service) SetDepth(ctx context.Context, courseID string, depth Depth) error {
	if err := s.validate(); err != nil {
		return err
	}
	if _, ok := depthRank(depth); !ok {
		return fmt.Errorf("invalid opening depth %q", depth)
	}
	course, err := s.catalog.LoadActive(ctx, courseID)
	if err != nil {
		return fmt.Errorf("load opening course %q: %w", courseID, err)
	}
	now := s.now().UTC()
	if err := s.store.SetDepth(ctx, courseID, depth, now); err != nil {
		return err
	}
	journey, err := s.store.Journey(ctx, courseID, depth)
	if err != nil {
		return err
	}
	if journey.CreatedAt.IsZero() {
		journey.CreatedAt = now
	}
	journey.Depth = depth
	journey.UpdatedAt = now
	resumable, err := s.store.ResumableSession(ctx)
	if err != nil {
		return err
	}
	projection, err := s.projectTeachingTree(ctx, course, depth, journey, resumable)
	if err != nil {
		return err
	}
	journey.LastRecommendedLessonID = projection.recommendedLessonID
	return s.store.SaveJourney(ctx, journey)
}

func (s *Service) Explore(
	ctx context.Context,
	courseID string,
	positionID string,
	depth Depth,
) (ExplorerPositionView, error) {
	if err := s.validate(); err != nil {
		return ExplorerPositionView{}, err
	}
	if _, ok := depthRank(depth); !ok {
		return ExplorerPositionView{}, fmt.Errorf("invalid opening depth %q", depth)
	}
	course, err := s.catalog.LoadActive(ctx, courseID)
	if err != nil {
		return ExplorerPositionView{}, fmt.Errorf("load opening course %q: %w", courseID, err)
	}
	position, exists := course.Positions[positionID]
	if !exists {
		return ExplorerPositionView{}, fmt.Errorf(
			"opening position %q was not found in course %q", positionID, courseID,
		)
	}
	view := ExplorerPositionView{
		CourseID: courseID, PositionID: positionID, FEN: position.FEN,
		Label: position.Label, Evaluation: position.Evaluation,
		Notes: []NoteView{}, Moves: []ExplorerMove{},
	}
	for _, noteID := range position.NoteIDs {
		note, exists := course.Notes[noteID]
		if !exists {
			return ExplorerPositionView{}, fmt.Errorf(
				"opening position %q references missing note %q", positionID, noteID,
			)
		}
		view.Notes = append(view.Notes, NoteView{
			Kind: note.Kind, Text: note.Text, SourceRef: note.SourceRef,
		})
	}
	for _, move := range course.VisibleMoves(positionID, depth) {
		view.Moves = append(view.Moves, ExplorerMove{
			MoveID: move.MoveID, UCI: move.UCI, SAN: move.SAN,
			ToPositionID: move.ToPositionID, Role: move.TrainingRole,
			VariationName: move.VariationName, Evaluation: move.Evaluation,
			SourceRef: move.SourceRef,
		})
	}
	for _, moveID := range course.Incoming[positionID] {
		move, exists := course.Moves[moveID]
		if exists && visibleAtDepth(move.MinimumDepth, depth) {
			view.IncomingPaths++
		}
	}
	return view, nil
}

func (s *Service) StartLesson(
	ctx context.Context,
	courseID string,
	lessonID string,
) (OpeningSessionView, error) {
	if err := s.validate(); err != nil {
		return OpeningSessionView{}, err
	}
	generationID, course, err := s.loadActiveCourse(ctx, courseID)
	if err != nil {
		return OpeningSessionView{}, err
	}
	depth, err := s.store.Depth(ctx, courseID, course.Pack.DefaultDepth)
	if err != nil {
		return OpeningSessionView{}, err
	}
	lesson, exists := course.Lessons[lessonID]
	if !exists {
		return OpeningSessionView{}, fmt.Errorf("opening lesson %q was not found", lessonID)
	}
	if !visibleAtDepth(lesson.MinimumDepth, depth) {
		return OpeningSessionView{}, fmt.Errorf("opening lesson %q is hidden at %s depth", lessonID, depth)
	}
	if len(lesson.Steps) == 0 {
		return OpeningSessionView{}, fmt.Errorf("opening lesson %q has no steps", lessonID)
	}
	now := s.now().UTC()
	state, err := s.stateForLessonStep(course, lesson.Steps[0], SessionState{
		Position: PositionState{PlayedMoveIDs: []string{}},
	}, now)
	if err != nil {
		return OpeningSessionView{}, err
	}
	session, err := s.store.CreateSession(ctx, SessionSeed{
		CourseID: courseID, GenerationID: generationID, LessonID: lessonID,
		Mode: OpeningModeLesson, Depth: depth, State: state,
	}, now)
	if err != nil {
		return OpeningSessionView{}, err
	}
	return s.sessionView(course, session)
}

func (s *Service) Resume(ctx context.Context) (*OpeningSessionView, error) {
	return s.resume(ctx)
}

func (s *Service) Pause(ctx context.Context, sessionID string) error {
	if err := s.validate(); err != nil {
		return err
	}
	session, err := s.store.LoadSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if session.Status != OpeningStatusActive {
		return fmt.Errorf("opening session %q is not active", sessionID)
	}
	return s.store.SetSessionStatus(ctx, sessionID, OpeningStatusPaused, s.now().UTC())
}

func (s *Service) StartReview(ctx context.Context, courseID string) (OpeningSessionView, error) {
	if err := s.validate(); err != nil {
		return OpeningSessionView{}, err
	}
	generationID, course, err := s.loadActiveCourse(ctx, courseID)
	if err != nil {
		return OpeningSessionView{}, err
	}
	depth, err := s.store.Depth(ctx, courseID, course.Pack.DefaultDepth)
	if err != nil {
		return OpeningSessionView{}, err
	}
	if err := s.applyCourseRevision(ctx, course, nil); err != nil {
		return OpeningSessionView{}, err
	}
	due, err := s.store.DueReviews(ctx, courseID, s.now().UTC(), 10000)
	if err != nil {
		return OpeningSessionView{}, err
	}
	promptIDs := make([]string, 0, len(due))
	for _, review := range due {
		prompt, exists := course.Prompts[review.PromptID]
		if exists && prompt.SemanticFingerprint == review.SemanticFingerprint &&
			promptVisibleAtDepth(course, prompt, depth) {
			promptIDs = append(promptIDs, review.PromptID)
		}
	}
	if len(promptIDs) == 0 {
		return OpeningSessionView{}, errors.New("no opening reviews are due")
	}
	now := s.now().UTC()
	state, err := s.stateForReviewPrompt(course, promptIDs[0], SessionState{
		Position: PositionState{PlayedMoveIDs: []string{}},
		Review:   &ReviewCursor{PromptIDs: promptIDs},
	}, now)
	if err != nil {
		return OpeningSessionView{}, err
	}
	session, err := s.store.CreateSession(ctx, SessionSeed{
		CourseID: courseID, GenerationID: generationID, LessonID: reviewLessonID,
		Mode: OpeningModeReview, Depth: depth, State: state,
	}, now)
	if err != nil {
		return OpeningSessionView{}, err
	}
	return s.sessionView(course, session)
}

func (s *Service) loadActiveCourse(
	ctx context.Context,
	courseID string,
) (string, CompiledCourse, error) {
	generationID, err := s.catalog.ActiveGenerationID(ctx, courseID)
	if err != nil {
		return "", CompiledCourse{}, fmt.Errorf("load active opening course %q: %w", courseID, err)
	}
	course, err := s.catalog.LoadGeneration(ctx, generationID)
	if err != nil {
		return "", CompiledCourse{}, fmt.Errorf("load opening course generation: %w", err)
	}
	return generationID, course, nil
}

func (s *Service) validate() error {
	if s == nil || s.catalog == nil || s.store == nil || s.rules == nil || s.now == nil {
		return errors.New("opening service is unavailable")
	}
	return nil
}

func visibleAtDepth(minimum Depth, selected Depth) bool {
	minimumRank, minimumOK := depthRank(minimum)
	selectedRank, selectedOK := depthRank(selected)
	return minimumOK && selectedOK && minimumRank <= selectedRank
}

func promptVisibleAtDepth(course CompiledCourse, prompt CompiledPrompt, depth Depth) bool {
	move, exists := course.Moves[prompt.PrimaryMoveID]
	return exists && visibleAtDepth(move.MinimumDepth, depth)
}

func lessonStepIDs(lesson Lesson) []string {
	ids := make([]string, len(lesson.Steps))
	for index, step := range lesson.Steps {
		ids[index] = step.StepID
	}
	return ids
}

func nextAttemptID() string {
	return uuid.NewString()
}

func coursePromptFingerprints(course CompiledCourse) map[string]string {
	fingerprints := make(map[string]string, len(course.Prompts))
	for promptID, prompt := range course.Prompts {
		fingerprints[promptID] = prompt.SemanticFingerprint
	}
	return fingerprints
}
