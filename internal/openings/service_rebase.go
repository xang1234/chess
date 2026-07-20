package openings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const (
	updatedCourseNotice = "The private course was updated. Your learned progress is preserved; restart from the last compatible teaching checkpoint."
	removedLessonNotice = "The private course was updated and this lesson is no longer available. Restart at the first available lesson."
	missingCourseNotice = "The private course data for this session is unavailable. Reimport the private course pack to continue."
)

type SessionAwareMaintenance struct {
	Catalog *SQLiteCatalog
	Store   *UserStore
}

func (s *Service) applyCourseRevision(
	ctx context.Context,
	course CompiledCourse,
	rebase *SessionRebase,
	journey *CourseJourney,
) error {
	return s.store.ApplyCourseRevision(ctx, CourseRevision{
		CourseID:           course.Pack.CourseID,
		PromptFingerprints: coursePromptFingerprints(course),
		SessionRebase:      rebase,
		Journey:            journey,
		Now:                s.now().UTC(),
	})
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
	if session.Status == OpeningStatusRestartRequired {
		if err := s.applyCourseRevision(ctx, course, nil, nil); err != nil {
			return nil, err
		}
		view := restartRequiredView(session, updatedCourseNotice)
		return &view, nil
	}
	if session.Status == OpeningStatusPaused {
		session.Status = OpeningStatusActive
		if err := s.applyCourseRevision(ctx, course, &SessionRebase{
			PreviousGenerationID: session.GenerationID,
			Session:              session,
		}, nil); err != nil {
			return nil, err
		}
	} else if err := s.applyCourseRevision(ctx, course, nil, nil); err != nil {
		return nil, err
	}
	view, err := s.sessionView(ctx, course, session)
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
	now := s.now().UTC()
	journey, err := s.store.Journey(ctx, session.CourseID)
	if err != nil {
		return nil, err
	}
	if journey.CreatedAt.IsZero() {
		journey.CreatedAt = now
	}
	journey.CurrentLessonID = session.LessonID
	journey.UpdatedAt = now
	if newExists {
		journey.PathLessonIDs = teachingPathLessonIDs(newCourse, session.LessonID)
	}

	if oldExists && newExists && session.ActivityIndex >= 0 && session.ActivityIndex < len(oldLesson.Activities) {
		oldActivity := oldLesson.Activities[session.ActivityIndex]
		if newIndex, newActivity, exists := lessonActivityByID(newLesson, oldActivity.ActivityID); exists &&
			activityCompatible(oldCourse, oldActivity, newCourse, newActivity) &&
			activityStatePositionCompatible(oldCourse, oldActivity, newCourse, newActivity, session.State.Position.PositionID) &&
			playedMovesCompatible(oldCourse, newCourse, session.State.Position.PlayedMoveIDs) {
			previousGenerationID := session.GenerationID
			session.GenerationID = activeGenerationID
			session.Status = OpeningStatusActive
			session.ActivityIndex = newIndex
			if newActivity.PositionID != "" {
				session.State.Position.PositionID = newActivity.PositionID
				session.State.Position.CurrentFEN = newCourse.Positions[newActivity.PositionID].FEN
			}
			if session.State.Attempt != nil {
				session.State.Attempt.PromptID = newActivity.PromptID
			}
			session.State.Restart = nil
			if err := s.applyCourseRevision(ctx, newCourse, &SessionRebase{
				PreviousGenerationID: previousGenerationID,
				Session:              session,
			}, &journey); err != nil {
				return nil, err
			}
			view, err := s.sessionView(ctx, newCourse, session)
			return &view, err
		}
	}

	session.Status = OpeningStatusRestartRequired
	session.State.Attempt = nil
	session.State.Restart = compatibleActivityCheckpoint(
		oldCourse, oldLesson, newCourse, newLesson, session.ActivityIndex,
	)
	if err := s.applyCourseRevision(ctx, newCourse, &SessionRebase{
		PreviousGenerationID: session.GenerationID,
		Session:              session,
	}, &journey); err != nil {
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
	if session.State.Review == nil {
		return nil, errors.New("review session requires a review cursor")
	}
	oldQueue := append([]string{}, session.State.Review.PromptIDs...)
	start := session.State.Review.Index
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
	session.State.Review.PromptIDs = pending
	session.State.Review.Index = 0
	session.ActivityIndex = 0
	session.State.Restart = nil
	if len(pending) == 0 {
		session.Status = OpeningStatusCompleted
		session.State = resetAttemptState(session.State)
		if err := s.applyCourseRevision(ctx, newCourse, &SessionRebase{
			PreviousGenerationID: previousGenerationID,
			Session:              session,
		}, nil); err != nil {
			return nil, err
		}
		view, err := s.sessionView(ctx, newCourse, session)
		view.Notice = "The private course was updated; no queued review positions remain."
		return &view, err
	}
	preserveAttempt := start < len(oldQueue) && oldQueue[start] == pending[0]
	if preserveAttempt {
		prompt := newCourse.Prompts[pending[0]]
		session.State.Position.PositionID = prompt.PositionID
		session.State.Position.CurrentFEN = newCourse.Positions[prompt.PositionID].FEN
	} else {
		var err error
		session.State, err = s.stateForReviewPrompt(newCourse, pending[0], session.State, s.now().UTC())
		if err != nil {
			return nil, err
		}
	}
	session.Status = OpeningStatusActive
	if err := s.applyCourseRevision(ctx, newCourse, &SessionRebase{
		PreviousGenerationID: previousGenerationID,
		Session:              session,
	}, nil); err != nil {
		return nil, err
	}
	view, err := s.sessionView(ctx, newCourse, session)
	return &view, err
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
		session.State.Restart = nil
	}
	activityIndex := -1
	if session.State.Restart != nil && session.State.Restart.ActivityIndex < len(lesson.Activities) {
		activityIndex = session.State.Restart.ActivityIndex
	}
	if activityIndex < 0 {
		progress, progressErr := s.store.LessonProgress(
			ctx, session.CourseID, lesson.LessonID, RequiredActivityIDs(lesson),
		)
		if progressErr != nil {
			return OpeningSessionView{}, progressErr
		}
		var found bool
		activityIndex, found = firstStudyActivityIndex(lesson, progress)
		if !found {
			return OpeningSessionView{}, fmt.Errorf("opening lesson %q has no study activity", lesson.LessonID)
		}
	}
	activity := lesson.Activities[activityIndex]
	startPosition, exists := course.Positions[lesson.StartPositionID]
	if !exists {
		return OpeningSessionView{}, fmt.Errorf("opening lesson %q start position is unavailable", lesson.LessonID)
	}
	now := s.now().UTC()
	state, err := s.stateForActivity(course, activity, SessionState{
		Position: PositionState{
			PositionID: lesson.StartPositionID, CurrentFEN: startPosition.FEN,
			PlayedMoveIDs: []string{},
		},
	}, now)
	if err != nil {
		return OpeningSessionView{}, err
	}
	session.GenerationID = activeGenerationID
	session.Status = OpeningStatusActive
	session.ActivityIndex = activityIndex
	session.State = state
	journey, err := s.store.Journey(ctx, session.CourseID)
	if err != nil {
		return OpeningSessionView{}, err
	}
	if journey.CreatedAt.IsZero() {
		journey.CreatedAt = now
	}
	journey.CurrentLessonID = lesson.LessonID
	journey.PathLessonIDs = teachingPathLessonIDs(course, lesson.LessonID)
	journey.UpdatedAt = now
	if err := s.applyCourseRevision(ctx, course, &SessionRebase{
		PreviousGenerationID: previousGenerationID,
		Session:              session,
	}, &journey); err != nil {
		return OpeningSessionView{}, err
	}
	return s.sessionView(ctx, course, session)
}

func (s *Service) restartReview(
	ctx context.Context,
	session StoredSession,
	previousGenerationID string,
	activeGenerationID string,
	course CompiledCourse,
) (OpeningSessionView, error) {
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
		Position: PositionState{PlayedMoveIDs: []string{}},
		Review:   &ReviewCursor{PromptIDs: queue},
	}, s.now().UTC())
	if err != nil {
		return OpeningSessionView{}, err
	}
	session.GenerationID = activeGenerationID
	session.Status = OpeningStatusActive
	session.ActivityIndex = 0
	session.State = state
	if err := s.applyCourseRevision(ctx, course, &SessionRebase{
		PreviousGenerationID: previousGenerationID,
		Session:              session,
	}, nil); err != nil {
		return OpeningSessionView{}, err
	}
	return s.sessionView(ctx, course, session)
}

func (s *Service) requirePrivateReimport(
	ctx context.Context,
	session StoredSession,
) (*OpeningSessionView, error) {
	session.Status = OpeningStatusRestartRequired
	session.State.Attempt = nil
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
