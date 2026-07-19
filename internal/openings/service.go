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
	depth Depth,
	resumable *StoredSession,
) (OpeningCourseSummary, error) {
	if err := s.store.ReconcileReviews(ctx, course.Pack.CourseID, coursePromptFingerprints(course)); err != nil {
		return OpeningCourseSummary{}, fmt.Errorf("reconcile opening reviews: %w", err)
	}
	view := OpeningCourseSummary{
		CourseID: course.Pack.CourseID, Title: course.Pack.Title,
		Perspective: course.Pack.Perspective, Depth: depth,
		RootPositionID: course.Pack.RootPositionID,
		Chapters:       []OpeningChapterSummary{},
	}
	if resumable != nil && resumable.CourseID == course.Pack.CourseID {
		view.HasResumable = true
	}
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
			stepIDs := lessonStepIDs(lesson)
			progress, err := s.store.LessonProgress(ctx, course.Pack.CourseID, lesson.LessonID, stepIDs)
			if err != nil {
				return OpeningCourseSummary{}, fmt.Errorf("load progress for lesson %q: %w", lesson.LessonID, err)
			}
			lessonView := OpeningLessonSummary{
				LessonID: lesson.LessonID, Title: lesson.Title,
				CompletedSteps: progress.CompletedSteps,
				TotalSteps:     progress.TotalSteps,
				Completed:      progress.Completed,
			}
			chapterView.Lessons = append(chapterView.Lessons, lessonView)
			view.TotalLessons++
			if progress.Completed {
				view.CompletedLessons++
			} else if view.NextLessonID == "" {
				view.NextLessonID = lesson.LessonID
				view.NextLessonTitle = lesson.Title
			}
		}
		view.Chapters = append(view.Chapters, chapterView)
	}
	due, err := s.store.DueReviews(ctx, course.Pack.CourseID, s.now().UTC(), 10000)
	if err != nil {
		return OpeningCourseSummary{}, fmt.Errorf("load due opening reviews: %w", err)
	}
	for _, review := range due {
		prompt, exists := course.Prompts[review.PromptID]
		if exists && prompt.SemanticFingerprint == review.SemanticFingerprint &&
			promptVisibleAtDepth(course, prompt, depth) {
			view.DueReviews++
		}
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
	if _, err := s.catalog.LoadActive(ctx, courseID); err != nil {
		return fmt.Errorf("load opening course %q: %w", courseID, err)
	}
	return s.store.SetDepth(ctx, courseID, depth, s.now().UTC())
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
		PlayedMoveIDs: []string{},
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
		PlayedMoveIDs:   []string{},
		ReviewPromptIDs: promptIDs,
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
