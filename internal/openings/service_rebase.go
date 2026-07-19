package openings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const (
	updatedCourseNotice = "The private course was updated. Restart from the last compatible teaching checkpoint."
	removedLessonNotice = "The private course was updated and this lesson is no longer available. Restart at the first available lesson."
	missingCourseNotice = "The private course data for this session is unavailable. Reimport the private course pack to continue."
)

type SessionAwareMaintenance struct {
	Catalog *SQLiteCatalog
	Store   *UserStore
}

func (m SessionAwareMaintenance) CleanupBatch(ctx context.Context, limit int) (bool, error) {
	if m.Catalog == nil || m.Store == nil {
		return false, errors.New("session-aware course maintenance is unavailable")
	}
	protected, err := m.Store.ProtectedGenerationIDs(ctx)
	if err != nil {
		return false, err
	}
	return m.Catalog.CleanupBatch(ctx, protected, limit)
}

func (s *Service) resume(ctx context.Context) (*OpeningSessionView, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	session, err := s.store.ResumableSession(ctx)
	if err != nil || session == nil {
		return nil, err
	}
	activeGenerationID, err := s.catalog.ActiveGenerationID(ctx, session.CourseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return s.requirePrivateReimport(ctx, *session)
		}
		return nil, err
	}
	if activeGenerationID == session.GenerationID {
		return s.resumeCurrentGeneration(ctx, *session)
	}

	oldCourse, err := s.catalog.LoadGeneration(ctx, session.GenerationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return s.requirePrivateReimport(ctx, *session)
		}
		return nil, err
	}
	newCourse, err := s.catalog.LoadGeneration(ctx, activeGenerationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return s.requirePrivateReimport(ctx, *session)
		}
		return nil, err
	}
	if session.Mode == OpeningModeReview {
		return s.rebaseReview(ctx, *session, oldCourse, newCourse, activeGenerationID)
	}
	return s.rebaseLesson(ctx, *session, oldCourse, newCourse, activeGenerationID)
}

func (s *Service) resumeCurrentGeneration(
	ctx context.Context,
	session StoredSession,
) (*OpeningSessionView, error) {
	course, err := s.catalog.LoadGeneration(ctx, session.GenerationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return s.requirePrivateReimport(ctx, session)
		}
		return nil, err
	}
	if err := s.store.ReconcileReviews(ctx, session.CourseID, coursePromptFingerprints(course)); err != nil {
		return nil, err
	}
	if session.Status == OpeningStatusRestartRequired {
		view := restartRequiredView(session, updatedCourseNotice)
		return &view, nil
	}
	if session.Status == OpeningStatusPaused {
		session.Status = OpeningStatusActive
		if err := s.store.SaveSession(ctx, session, s.now().UTC()); err != nil {
			return nil, err
		}
	}
	view, err := s.sessionView(course, session)
	return &view, err
}

func (s *Service) rebaseLesson(
	ctx context.Context,
	session StoredSession,
	oldCourse CompiledCourse,
	newCourse CompiledCourse,
	activeGenerationID string,
) (*OpeningSessionView, error) {
	oldLesson, oldExists := oldCourse.Lessons[session.LessonID]
	newLesson, newExists := newCourse.Lessons[session.LessonID]
	if oldExists && newExists && session.StepIndex >= 0 && session.StepIndex < len(oldLesson.Steps) {
		oldStep := oldLesson.Steps[session.StepIndex]
		if newIndex, newStep, exists := lessonStepByID(newLesson, oldStep.StepID); exists &&
			oldStep.Kind == newStep.Kind &&
			sameStepPosition(oldCourse, oldStep, newCourse, newStep) &&
			playedMovesCompatible(oldCourse, newCourse, session.State.PlayedMoveIDs) &&
			promptStepCompatible(oldCourse, oldStep, newCourse, newStep) {
			previousGenerationID := session.GenerationID
			session.GenerationID = activeGenerationID
			session.Status = OpeningStatusActive
			session.StepIndex = newIndex
			session.State.PositionID = newStep.PositionID
			session.State.CurrentFEN = newCourse.Positions[newStep.PositionID].FEN
			session.State.RestartStepIndex = nil
			if err := s.store.RebaseSession(ctx, previousGenerationID, session, s.now().UTC()); err != nil {
				return nil, err
			}
			if err := s.store.ReconcileReviews(ctx, session.CourseID, coursePromptFingerprints(newCourse)); err != nil {
				return nil, err
			}
			view, err := s.sessionView(newCourse, session)
			return &view, err
		}
	}

	session.Status = OpeningStatusRestartRequired
	session.State.RestartStepIndex = compatibleCheckpoint(
		oldCourse, oldLesson, newCourse, newLesson, session.StepIndex,
	)
	if err := s.store.SaveSession(ctx, session, s.now().UTC()); err != nil {
		return nil, err
	}
	notice := updatedCourseNotice
	if !newExists {
		notice = removedLessonNotice
	}
	view := restartRequiredView(session, notice)
	return &view, nil
}

func (s *Service) rebaseReview(
	ctx context.Context,
	session StoredSession,
	oldCourse CompiledCourse,
	newCourse CompiledCourse,
	activeGenerationID string,
) (*OpeningSessionView, error) {
	oldQueue := append([]string{}, session.State.ReviewPromptIDs...)
	start := session.State.ReviewIndex
	if start < 0 {
		start = 0
	}
	if start > len(oldQueue) {
		start = len(oldQueue)
	}
	pending := []string{}
	for _, promptID := range oldQueue[start:] {
		oldPrompt, oldExists := oldCourse.Prompts[promptID]
		newPrompt, newExists := newCourse.Prompts[promptID]
		if oldExists && newExists && oldPrompt.SemanticFingerprint == newPrompt.SemanticFingerprint {
			pending = append(pending, promptID)
		}
	}
	previousGenerationID := session.GenerationID
	session.GenerationID = activeGenerationID
	session.State.ReviewPromptIDs = pending
	session.State.ReviewIndex = 0
	session.StepIndex = 0
	session.State.RestartStepIndex = nil
	if len(pending) == 0 {
		session.Status = OpeningStatusCompleted
		session.State = resetAttemptState(session.State)
		if err := s.persistReviewRebase(ctx, previousGenerationID, session, newCourse); err != nil {
			return nil, err
		}
		view, err := s.sessionView(newCourse, session)
		view.Notice = "The private course was updated; no queued review positions remain."
		return &view, err
	}
	preserveAttempt := start < len(oldQueue) && oldQueue[start] == pending[0]
	if preserveAttempt {
		prompt := newCourse.Prompts[pending[0]]
		session.State.PositionID = prompt.PositionID
		session.State.CurrentFEN = newCourse.Positions[prompt.PositionID].FEN
	} else {
		var err error
		session.State, err = s.stateForReviewPrompt(newCourse, pending[0], session.State, s.now().UTC())
		if err != nil {
			return nil, err
		}
	}
	session.Status = OpeningStatusActive
	if err := s.persistReviewRebase(ctx, previousGenerationID, session, newCourse); err != nil {
		return nil, err
	}
	view, err := s.sessionView(newCourse, session)
	return &view, err
}

func (s *Service) persistReviewRebase(
	ctx context.Context,
	previousGenerationID string,
	session StoredSession,
	course CompiledCourse,
) error {
	if err := s.store.RebaseSession(ctx, previousGenerationID, session, s.now().UTC()); err != nil {
		return err
	}
	return s.store.ReconcileReviews(ctx, session.CourseID, coursePromptFingerprints(course))
}

func (s *Service) Restart(ctx context.Context, sessionID string) (OpeningSessionView, error) {
	if err := s.validate(); err != nil {
		return OpeningSessionView{}, err
	}
	session, err := s.store.LoadSession(ctx, sessionID)
	if err != nil {
		return OpeningSessionView{}, err
	}
	if session.Status != OpeningStatusRestartRequired {
		return OpeningSessionView{}, fmt.Errorf("opening session %q does not require restart", sessionID)
	}
	activeGenerationID, course, err := s.loadActiveCourse(ctx, session.CourseID)
	if err != nil {
		return OpeningSessionView{}, fmt.Errorf("reimport the private course pack before restarting: %w", err)
	}
	previousGenerationID := session.GenerationID
	if session.Mode == OpeningModeReview {
		return s.restartReview(ctx, session, previousGenerationID, activeGenerationID, course)
	}
	lesson, exists := course.Lessons[session.LessonID]
	if !exists || !visibleAtDepth(lesson.MinimumDepth, session.Depth) {
		lesson, exists = firstVisibleLesson(course, session.Depth)
		if !exists {
			return OpeningSessionView{}, errors.New("the updated private course has no lesson at the selected depth")
		}
		session.LessonID = lesson.LessonID
		session.State.RestartStepIndex = nil
	}
	stepIndex := 0
	if session.State.RestartStepIndex != nil && *session.State.RestartStepIndex < len(lesson.Steps) {
		stepIndex = *session.State.RestartStepIndex
	}
	state, err := s.stateForLessonStep(course, lesson.Steps[stepIndex], SessionState{
		PlayedMoveIDs: []string{},
	}, s.now().UTC())
	if err != nil {
		return OpeningSessionView{}, err
	}
	session.GenerationID = activeGenerationID
	session.Status = OpeningStatusActive
	session.StepIndex = stepIndex
	session.State = state
	if err := s.store.RebaseSession(ctx, previousGenerationID, session, s.now().UTC()); err != nil {
		return OpeningSessionView{}, err
	}
	if err := s.store.ReconcileReviews(ctx, session.CourseID, coursePromptFingerprints(course)); err != nil {
		return OpeningSessionView{}, err
	}
	return s.sessionView(course, session)
}

func (s *Service) restartReview(
	ctx context.Context,
	session StoredSession,
	previousGenerationID string,
	activeGenerationID string,
	course CompiledCourse,
) (OpeningSessionView, error) {
	if err := s.store.ReconcileReviews(ctx, session.CourseID, coursePromptFingerprints(course)); err != nil {
		return OpeningSessionView{}, err
	}
	due, err := s.store.DueReviews(ctx, session.CourseID, s.now().UTC(), 10000)
	if err != nil {
		return OpeningSessionView{}, err
	}
	queue := []string{}
	for _, review := range due {
		prompt, exists := course.Prompts[review.PromptID]
		if exists && prompt.SemanticFingerprint == review.SemanticFingerprint &&
			promptVisibleAtDepth(course, prompt, session.Depth) {
			queue = append(queue, review.PromptID)
		}
	}
	if len(queue) == 0 {
		return OpeningSessionView{}, errors.New("the updated private course has no due review positions")
	}
	state, err := s.stateForReviewPrompt(course, queue[0], SessionState{
		PlayedMoveIDs: []string{}, ReviewPromptIDs: queue,
	}, s.now().UTC())
	if err != nil {
		return OpeningSessionView{}, err
	}
	session.GenerationID = activeGenerationID
	session.Status = OpeningStatusActive
	session.StepIndex = 0
	session.State = state
	if err := s.store.RebaseSession(ctx, previousGenerationID, session, s.now().UTC()); err != nil {
		return OpeningSessionView{}, err
	}
	return s.sessionView(course, session)
}

func (s *Service) requirePrivateReimport(
	ctx context.Context,
	session StoredSession,
) (*OpeningSessionView, error) {
	session.Status = OpeningStatusRestartRequired
	if err := s.store.SaveSession(ctx, session, s.now().UTC()); err != nil {
		return nil, err
	}
	view := restartRequiredView(session, missingCourseNotice)
	return &view, nil
}

func restartRequiredView(session StoredSession, notice string) OpeningSessionView {
	return OpeningSessionView{
		SessionID: session.ID, Mode: session.Mode, Status: OpeningStatusRestartRequired,
		CourseID: session.CourseID, GenerationID: session.GenerationID,
		LessonID: session.LessonID, Depth: session.Depth, Notice: notice,
	}
}

func lessonStepByID(lesson Lesson, stepID string) (int, LessonStep, bool) {
	for index, step := range lesson.Steps {
		if step.StepID == stepID {
			return index, step, true
		}
	}
	return 0, LessonStep{}, false
}

func sameStepPosition(
	oldCourse CompiledCourse,
	oldStep LessonStep,
	newCourse CompiledCourse,
	newStep LessonStep,
) bool {
	oldPosition, oldExists := oldCourse.Positions[oldStep.PositionID]
	newPosition, newExists := newCourse.Positions[newStep.PositionID]
	if !oldExists || !newExists {
		return false
	}
	oldCanonical, oldErr := CanonicalPosition(oldPosition.FEN)
	newCanonical, newErr := CanonicalPosition(newPosition.FEN)
	return oldErr == nil && newErr == nil && oldCanonical == newCanonical
}

func playedMovesCompatible(oldCourse CompiledCourse, newCourse CompiledCourse, moveIDs []string) bool {
	for _, moveID := range moveIDs {
		oldMove, oldExists := oldCourse.Moves[moveID]
		newMove, newExists := newCourse.Moves[moveID]
		if !oldExists || !newExists || oldMove.UCI != newMove.UCI ||
			!sameMovePosition(oldCourse, oldMove.FromPositionID, newCourse, newMove.FromPositionID) ||
			!sameMovePosition(oldCourse, oldMove.ToPositionID, newCourse, newMove.ToPositionID) {
			return false
		}
	}
	return true
}

func sameMovePosition(oldCourse CompiledCourse, oldID string, newCourse CompiledCourse, newID string) bool {
	oldPosition, oldExists := oldCourse.Positions[oldID]
	newPosition, newExists := newCourse.Positions[newID]
	if !oldExists || !newExists {
		return false
	}
	oldCanonical, oldErr := CanonicalPosition(oldPosition.FEN)
	newCanonical, newErr := CanonicalPosition(newPosition.FEN)
	return oldErr == nil && newErr == nil && oldCanonical == newCanonical
}

func promptStepCompatible(
	oldCourse CompiledCourse,
	oldStep LessonStep,
	newCourse CompiledCourse,
	newStep LessonStep,
) bool {
	if oldStep.PromptID == "" && newStep.PromptID == "" {
		return true
	}
	oldPrompt, oldExists := oldCourse.Prompts[oldStep.PromptID]
	newPrompt, newExists := newCourse.Prompts[newStep.PromptID]
	return oldExists && newExists && oldPrompt.SemanticFingerprint == newPrompt.SemanticFingerprint
}

func compatibleCheckpoint(
	oldCourse CompiledCourse,
	oldLesson Lesson,
	newCourse CompiledCourse,
	newLesson Lesson,
	currentIndex int,
) *int {
	if strings.TrimSpace(oldLesson.LessonID) == "" || strings.TrimSpace(newLesson.LessonID) == "" {
		return nil
	}
	if currentIndex >= len(oldLesson.Steps) {
		currentIndex = len(oldLesson.Steps) - 1
	}
	for index := currentIndex; index >= 0; index-- {
		oldStep := oldLesson.Steps[index]
		if oldStep.Kind != StepExplain && oldStep.Kind != StepWatch {
			continue
		}
		newIndex, newStep, exists := lessonStepByID(newLesson, oldStep.StepID)
		if exists && newStep.Kind == oldStep.Kind && sameStepPosition(oldCourse, oldStep, newCourse, newStep) {
			checkpoint := newIndex
			return &checkpoint
		}
	}
	return nil
}

func firstVisibleLesson(course CompiledCourse, depth Depth) (Lesson, bool) {
	for _, lesson := range course.Pack.Lessons {
		if visibleAtDepth(lesson.MinimumDepth, depth) && len(lesson.Steps) > 0 {
			return lesson, true
		}
	}
	return Lesson{}, false
}
