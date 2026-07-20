package openings

import (
	"context"
	"errors"
	"fmt"

	"chess-trainer/internal/domain"
)

func (s *Service) AdvanceActivity(ctx context.Context, sessionID string) (OpeningActivityResult, error) {
	session, course, lesson, activity, err := s.loadLessonActivity(ctx, sessionID)
	if err != nil {
		return OpeningActivityResult{}, err
	}
	switch activity.Kind {
	case ActivityConcept, ActivityComparison, ActivityRecap, ActivityReference:
	case ActivityDemonstration:
	default:
		return OpeningActivityResult{}, fmt.Errorf("%s opening activities require a move", activity.Kind)
	}
	frames := []domain.AppliedMove{}
	finalFEN := ""
	if activity.Kind == ActivityDemonstration {
		frames, finalFEN, err = s.applyMoveIDs(course, session.State.Position.CurrentFEN, activity.MoveIDs)
		if err != nil {
			return OpeningActivityResult{}, err
		}
		session.State.Position.PlayedMoveIDs = append(session.State.Position.PlayedMoveIDs, activity.MoveIDs...)
	}
	return s.completeLessonActivity(
		ctx, course, session, lesson, activity, nil, "", "", frames, finalFEN,
	)
}

func (s *Service) completeLessonActivity(
	ctx context.Context,
	course CompiledCourse,
	session StoredSession,
	lesson Lesson,
	activity LessonActivity,
	attempt *AttemptRecord,
	semanticFingerprint string,
	outcome ReviewOutcome,
	frames []domain.AppliedMove,
	finalFEN string,
) (OpeningActivityResult, error) {
	requiredIDs := RequiredActivityIDs(lesson)
	progress, err := s.store.LessonProgress(ctx, session.CourseID, session.LessonID, requiredIDs)
	if err != nil {
		return OpeningActivityResult{}, err
	}
	now := s.now().UTC()
	journey, err := s.store.Journey(ctx, session.CourseID)
	if err != nil {
		return OpeningActivityResult{}, err
	}
	if journey.CreatedAt.IsZero() {
		journey.CreatedAt = now
	}
	journey.CurrentLessonID = session.LessonID
	journey.PathLessonIDs = teachingPathLessonIDs(course, session.LessonID)
	journey.UpdatedAt = now

	completedSet := make(map[string]bool, len(progress.CompletedActivityIDs)+1)
	for _, activityID := range progress.CompletedActivityIDs {
		completedSet[activityID] = true
	}
	completedSet[activity.ActivityID] = true
	nextIndex, hasNext := nextStudyActivityIndex(lesson, session.ActivityIndex, completedSet, progress.Completed)
	var checkpoint *OpeningRoadmapCheckpoint
	if hasNext {
		session.ActivityIndex = nextIndex
		session.State, err = s.stateForActivity(course, lesson.Activities[nextIndex], session.State, now)
		if err != nil {
			return OpeningActivityResult{}, err
		}
	} else {
		session.Status = OpeningStatusCompleted
		if finalFEN != "" {
			session.State.Position.CurrentFEN = finalFEN
			if len(frames) != 0 {
				lastMoveID := append([]string{}, activity.MoveIDs...)
				if attempt != nil {
					prompt := course.Prompts[attempt.PromptID]
					lastMoveID = append([]string{prompt.PrimaryMoveID}, activity.MoveIDs...)
				}
				if len(lastMoveID) != 0 {
					session.State.Position.PositionID = course.Moves[lastMoveID[len(lastMoveID)-1]].ToPositionID
				}
			}
		}
		session.State.Attempt = nil
		checkpoint, err = s.lessonCheckpointAfterCompletion(ctx, course, session.LessonID, session.Depth, journey)
		if err != nil {
			return OpeningActivityResult{}, err
		}
	}

	if err := s.store.CompleteLessonActivity(ctx, LessonActivityCompletion{
		Session: session, Journey: journey, ActivityID: activity.ActivityID,
		RequiredActivityIDs: requiredIDs, Attempt: attempt,
		SemanticFingerprint: semanticFingerprint, Outcome: outcome, Now: now,
	}); err != nil {
		return OpeningActivityResult{}, err
	}
	view, err := s.sessionView(ctx, course, session)
	if err != nil {
		return OpeningActivityResult{}, err
	}
	return OpeningActivityResult{
		Session: view, ActivityCompleted: true, StepCompleted: true,
		Feedback: FeedbackExpected, AppliedMoves: frames, FinalFEN: finalFEN,
		Checkpoint: checkpoint,
	}, nil
}

func (s *Service) lessonCheckpointAfterCompletion(
	ctx context.Context,
	course CompiledCourse,
	lessonID string,
	depth Depth,
	journey CourseJourney,
) (*OpeningRoadmapCheckpoint, error) {
	projection, err := s.projectTeachingTree(ctx, course, depth, journey, nil)
	if err != nil {
		return nil, err
	}
	completedNow := !projection.progressByLesson[lessonID].Completed
	progress := projection.progressByLesson[lessonID]
	progress.Completed = true
	progress.CompletedActivities = progress.TotalActivities
	projection.progressByLesson[lessonID] = progress
	recommendedID := recommendTeachingLesson(course, depth, journey, nil, projection.progressByLesson)
	recommendedTitle := ""
	if recommendedID != "" {
		recommendedTitle = course.Lessons[recommendedID].Title
	}
	completedLessons := projection.completedLessons
	if completedNow && visibleAtDepth(course.Lessons[lessonID].MinimumDepth, depth) {
		completedLessons++
	}
	available := []string{}
	for _, edge := range course.LessonChildren[lessonID] {
		child, exists := course.Lessons[edge.ToLessonID]
		if exists && visibleAtDepth(child.MinimumDepth, depth) {
			available = append(available, child.LessonID)
		}
	}
	return &OpeningRoadmapCheckpoint{
		CompletedLessonID: lessonID, Path: openingPathItems(course, journey.PathLessonIDs),
		AvailableLessonIDs: available, RecommendedLessonID: recommendedID,
		RecommendedLessonTitle: recommendedTitle,
		CompletedLessons:       completedLessons, TotalLessons: projection.totalLessons,
	}, nil
}

func firstStudyActivityIndex(lesson Lesson, progress LessonProgress) (int, bool) {
	completed := make(map[string]bool, len(progress.CompletedActivityIDs))
	for _, activityID := range progress.CompletedActivityIDs {
		completed[activityID] = true
	}
	for index, activity := range lesson.Activities {
		if activity.Required && (progress.Completed || !completed[activity.ActivityID]) {
			return index, true
		}
	}
	for index, activity := range lesson.Activities {
		if activity.Required {
			return index, true
		}
	}
	return 0, false
}

func nextStudyActivityIndex(
	lesson Lesson,
	currentIndex int,
	completed map[string]bool,
	restudying bool,
) (int, bool) {
	if !restudying {
		for index, activity := range lesson.Activities {
			if activity.Required && !completed[activity.ActivityID] {
				return index, true
			}
		}
		return 0, false
	}
	for index := currentIndex + 1; index < len(lesson.Activities); index++ {
		activity := lesson.Activities[index]
		if activity.Required {
			return index, true
		}
	}
	return 0, false
}

func (s *Service) loadLessonActivity(
	ctx context.Context,
	sessionID string,
) (StoredSession, CompiledCourse, Lesson, LessonActivity, error) {
	session, err := s.store.LoadSession(ctx, sessionID)
	if err != nil {
		return StoredSession{}, CompiledCourse{}, Lesson{}, LessonActivity{}, err
	}
	if session.Status != OpeningStatusActive {
		return StoredSession{}, CompiledCourse{}, Lesson{}, LessonActivity{}, fmt.Errorf("opening session %q is not active", sessionID)
	}
	if session.Mode != OpeningModeLesson {
		return StoredSession{}, CompiledCourse{}, Lesson{}, LessonActivity{}, errors.New("review activities cannot be advanced without a move")
	}
	course, err := s.catalog.LoadGeneration(ctx, session.GenerationID)
	if err != nil {
		return StoredSession{}, CompiledCourse{}, Lesson{}, LessonActivity{}, err
	}
	lesson, exists := course.Lessons[session.LessonID]
	if !exists || session.ActivityIndex < 0 || session.ActivityIndex >= len(lesson.Activities) {
		return StoredSession{}, CompiledCourse{}, Lesson{}, LessonActivity{}, errors.New("opening session lesson activity is unavailable")
	}
	return session, course, lesson, lesson.Activities[session.ActivityIndex], nil
}

func (s *Service) sessionView(
	ctx context.Context,
	course CompiledCourse,
	session StoredSession,
) (OpeningSessionView, error) {
	view := OpeningSessionView{
		SessionID: session.ID, Mode: session.Mode, Status: session.Status,
		CourseID: session.CourseID, GenerationID: session.GenerationID,
		LessonID: session.LessonID, CourseTitle: course.Pack.Title,
		Path:  openingPathItems(course, teachingPathLessonIDs(course, session.LessonID)),
		Depth: session.Depth,
	}
	if session.Status == OpeningStatusCompleted {
		if session.Mode == OpeningModeReview {
			view.Summary = openingSummary(session.State)
		}
		return view, nil
	}
	var current OpeningActivityView
	var err error
	if session.Mode == OpeningModeLesson {
		lesson, exists := course.Lessons[session.LessonID]
		if !exists || session.ActivityIndex < 0 || session.ActivityIndex >= len(lesson.Activities) {
			return OpeningSessionView{}, errors.New("opening session lesson activity is unavailable")
		}
		current, err = s.lessonActivityView(ctx, course, lesson, session)
	} else {
		current, err = s.reviewActivityView(course, session)
	}
	if err != nil {
		return OpeningSessionView{}, err
	}
	view.Current = &current
	return view, nil
}

func (s *Service) lessonActivityView(
	ctx context.Context,
	course CompiledCourse,
	lesson Lesson,
	session StoredSession,
) (OpeningActivityView, error) {
	activity := lesson.Activities[session.ActivityIndex]
	requiredIDs := RequiredActivityIDs(lesson)
	progress, err := s.store.LessonProgress(ctx, session.CourseID, session.LessonID, requiredIDs)
	if err != nil {
		return OpeningActivityView{}, err
	}
	number := 0
	for index, candidate := range lesson.Activities {
		if candidate.Required {
			number++
		}
		if index == session.ActivityIndex {
			break
		}
	}
	return s.buildActivityView(course, lesson, activity, session, number, len(requiredIDs), progress.CompletedActivities)
}

func (s *Service) buildActivityView(
	course CompiledCourse,
	lesson Lesson,
	activity LessonActivity,
	session StoredSession,
	number int,
	total int,
	completed int,
) (OpeningActivityView, error) {
	positionID := activity.PositionID
	if positionID == "" {
		positionID = session.State.Position.PositionID
	}
	position, exists := course.Positions[positionID]
	if !exists {
		return OpeningActivityView{}, fmt.Errorf("opening position %q is unavailable", positionID)
	}
	teachingNotes := noteTextsForIDs(course, activity.NoteIDs, nil)
	movesToHere, err := s.movesToPosition(course, positionID, session.Depth)
	if err != nil {
		return OpeningActivityView{}, err
	}
	view := OpeningActivityView{
		ActivityID: activity.ActivityID, Kind: activity.Kind, Title: activity.Title,
		Instruction: activity.Instruction, Required: activity.Required, PositionID: positionID,
		CurrentFEN: session.State.Position.CurrentFEN, Orientation: course.Pack.Perspective,
		LegalMoves: []string{}, TeachingNoteTexts: teachingNotes,
		ReferenceNoteTexts: s.activityReferenceNoteTexts(course, activity, teachingNotes),
		Comparison:         activityComparisonLines(course, activity.Comparison),
		Annotations:        append([]BoardAnnotation{}, activity.Annotations...),
		MovesToHere:        movesToHere, ActivityNumber: number, ActivityTotal: total,
		CompletedIdeas: completed, RequiredIdeas: total,
		ReferenceSections: referenceSections(course, lesson),
	}
	if view.CurrentFEN == "" {
		view.CurrentFEN = position.FEN
	}
	if session.State.Attempt != nil {
		view.HintLevel = session.State.Attempt.HintLevel
		view.CanReveal = session.State.Attempt.HintLevel >= 4
	}
	if activity.PromptID != "" {
		prompt := course.Prompts[activity.PromptID]
		primary := course.Moves[prompt.PrimaryMoveID]
		view.VariationName = primary.VariationName
		view.LegalMoves, err = s.rules.LegalMoves(view.CurrentFEN)
		if err != nil {
			return OpeningActivityView{}, fmt.Errorf("list opening moves: %w", err)
		}
	} else if len(activity.MoveIDs) != 0 {
		view.VariationName = course.Moves[activity.MoveIDs[len(activity.MoveIDs)-1]].VariationName
	}
	return view, nil
}

func activityComparisonLines(course CompiledCourse, lines []ActivityLine) []OpeningActivityLine {
	result := make([]OpeningActivityLine, 0, len(lines))
	for _, line := range lines {
		moves := make([]string, 0, len(line.MoveIDs))
		for _, moveID := range line.MoveIDs {
			if move, ok := course.Moves[moveID]; ok {
				moves = append(moves, move.SAN)
			}
		}
		result = append(result, OpeningActivityLine{Label: line.Label, Moves: moves})
	}
	return result
}

func referenceSections(course CompiledCourse, lesson Lesson) []OpeningReferenceSection {
	sections := []OpeningReferenceSection{}
	for _, activity := range lesson.Activities {
		if activity.Kind != ActivityReference {
			continue
		}
		sections = append(sections, OpeningReferenceSection{
			ActivityID: activity.ActivityID, Title: activity.Title, Instruction: activity.Instruction,
			PositionID: activity.PositionID, NoteTexts: noteTextsForIDs(course, activity.NoteIDs, nil),
			Annotations: append([]BoardAnnotation{}, activity.Annotations...),
		})
	}
	return sections
}

func (s *Service) movesToPosition(
	course CompiledCourse,
	targetPositionID string,
	depth Depth,
) ([]domain.AppliedMove, error) {
	if targetPositionID == course.Pack.RootPositionID {
		return []domain.AppliedMove{}, nil
	}
	type route struct {
		positionID string
		moveIDs    []string
	}
	queue := []route{{positionID: course.Pack.RootPositionID, moveIDs: []string{}}}
	visited := map[string]bool{course.Pack.RootPositionID: true}
	for len(queue) != 0 {
		current := queue[0]
		queue = queue[1:]
		for _, moveID := range course.Outgoing[current.positionID] {
			move, exists := course.Moves[moveID]
			if !exists || !visibleAtDepth(move.MinimumDepth, depth) || visited[move.ToPositionID] {
				continue
			}
			path := append(append([]string{}, current.moveIDs...), moveID)
			if move.ToPositionID == targetPositionID {
				frames, _, err := s.applyMoveIDs(course, course.Pack.RootFEN, path)
				return frames, err
			}
			visited[move.ToPositionID] = true
			queue = append(queue, route{positionID: move.ToPositionID, moveIDs: path})
		}
	}
	return nil, fmt.Errorf("opening position %q is unreachable at %s depth", targetPositionID, depth)
}
