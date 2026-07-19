package openings

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"chess-trainer/internal/domain"
)

func (s *Service) Advance(ctx context.Context, sessionID string) (OpeningStepResult, error) {
	session, course, lesson, step, err := s.loadLessonStep(ctx, sessionID)
	if err != nil {
		return OpeningStepResult{}, err
	}
	if step.Kind != StepExplain && step.Kind != StepWatch {
		return OpeningStepResult{}, fmt.Errorf("%s opening steps require a move", step.Kind)
	}
	result := OpeningStepResult{StepCompleted: true}
	if step.Kind == StepWatch {
		frames, finalFEN, err := s.applyMoveIDs(course, course.Positions[step.PositionID].FEN, step.MoveIDs)
		if err != nil {
			return OpeningStepResult{}, err
		}
		result.AppliedMoves = frames
		result.FinalFEN = finalFEN
		session.State.Position.PlayedMoveIDs = append(session.State.Position.PlayedMoveIDs, step.MoveIDs...)
	}
	session.StepIndex++
	if session.StepIndex >= len(lesson.Steps) {
		return OpeningStepResult{}, errors.New("opening lesson ended without a recall step")
	}
	session.State, err = s.stateForLessonStep(course, lesson.Steps[session.StepIndex], session.State, s.now().UTC())
	if err != nil {
		return OpeningStepResult{}, err
	}
	if err := s.store.SaveSession(ctx, session, s.now().UTC()); err != nil {
		return OpeningStepResult{}, err
	}
	result.Session, err = s.sessionView(course, session)
	return result, err
}

func (s *Service) PlayMove(
	ctx context.Context,
	sessionID string,
	uci string,
) (OpeningStepResult, error) {
	session, course, step, prompt, err := s.loadPromptStep(ctx, sessionID)
	if err != nil {
		return OpeningStepResult{}, err
	}
	uci = strings.TrimSpace(uci)
	if uci == "" {
		return OpeningStepResult{}, errors.New("opening move is required")
	}
	if _, err := s.rules.ApplyUCI(session.State.Position.CurrentFEN, uci); err != nil {
		return OpeningStepResult{}, fmt.Errorf("illegal opening move %q: %w", uci, err)
	}
	primary := course.Moves[prompt.PrimaryMoveID]
	if uci == primary.UCI {
		return s.completePrimary(ctx, course, session, step, prompt)
	}
	for _, moveID := range prompt.AcceptedAlternativeMoveIDs {
		move, exists := course.Moves[moveID]
		if exists && visibleAtDepth(move.MinimumDepth, session.Depth) && move.UCI == uci {
			session.State.Attempt.AlternativesTried++
			if err := s.store.SaveSession(ctx, session, s.now().UTC()); err != nil {
				return OpeningStepResult{}, err
			}
			view, err := s.sessionView(course, session)
			return OpeningStepResult{
				Session: view, Feedback: FeedbackAlternative,
				Message: "That is a playable course alternative. Return to the lesson position and try the course move.",
			}, err
		}
	}
	session.State.Attempt.IncorrectMoves++
	if err := s.store.SaveSession(ctx, session, s.now().UTC()); err != nil {
		return OpeningStepResult{}, err
	}
	view, err := s.sessionView(course, session)
	return OpeningStepResult{
		Session:  view,
		Feedback: FeedbackOffCourse,
		Message:  fmt.Sprintf("That move is playable, but this lesson is practicing %s.", primary.SAN),
	}, err
}

func (s *Service) UseHint(ctx context.Context, sessionID string) (OpeningHintResult, error) {
	session, course, _, prompt, err := s.loadPromptStep(ctx, sessionID)
	if err != nil {
		return OpeningHintResult{}, err
	}
	if session.State.Attempt.HintLevel < 4 {
		session.State.Attempt.HintLevel++
		session.State.Attempt.HintsUsed++
		if err := s.store.SaveSession(ctx, session, s.now().UTC()); err != nil {
			return OpeningHintResult{}, err
		}
	}
	primary := course.Moves[prompt.PrimaryMoveID]
	result := OpeningHintResult{Level: session.State.Attempt.HintLevel, CanReveal: session.State.Attempt.HintLevel >= 4}
	switch session.State.Attempt.HintLevel {
	case 1:
		result.Text = s.planHint(course, prompt)
	case 2:
		result.Text = fmt.Sprintf("The course move starts on %s.", primary.UCI[:2])
		result.SourceSquare = primary.UCI[:2]
	case 3:
		result.Text = fmt.Sprintf("The course move lands on %s.", primary.UCI[2:4])
		result.SourceSquare = primary.UCI[:2]
		result.TargetSquare = primary.UCI[2:4]
	default:
		result.Text = "Show the course move."
		result.SourceSquare = primary.UCI[:2]
		result.TargetSquare = primary.UCI[2:4]
	}
	result.Session, err = s.sessionView(course, session)
	return result, err
}

func (s *Service) Reveal(ctx context.Context, sessionID string) (OpeningStepResult, error) {
	session, course, step, prompt, err := s.loadPromptStep(ctx, sessionID)
	if err != nil {
		return OpeningStepResult{}, err
	}
	if session.State.Attempt.HintLevel < 4 {
		return OpeningStepResult{}, errors.New("use all opening hints before revealing the course move")
	}
	session.State.Attempt.Revealed = true
	return s.completePrimary(ctx, course, session, step, prompt)
}

func (s *Service) completePrimary(
	ctx context.Context,
	course CompiledCourse,
	session StoredSession,
	step LessonStep,
	prompt CompiledPrompt,
) (OpeningStepResult, error) {
	primary := course.Moves[prompt.PrimaryMoveID]
	moveIDs := append([]string{primary.MoveID}, step.MoveIDs...)
	finalPositionID := course.Moves[moveIDs[len(moveIDs)-1]].ToPositionID
	frames, finalFEN, err := s.applyMoveIDs(course, session.State.Position.CurrentFEN, moveIDs)
	if err != nil {
		return OpeningStepResult{}, err
	}
	attempt, err := attemptRecord(session.State.Attempt)
	if err != nil {
		return OpeningStepResult{}, err
	}
	session.State.Position.PlayedMoveIDs = append(session.State.Position.PlayedMoveIDs, moveIDs...)
	session.State.Summary.CompletedPrompts++
	if step.Kind == StepBranch {
		session.State.Summary.BranchesRecognized++
	} else {
		session.State.Summary.PositionsRecalled++
	}
	if attempt.IncorrectMoves > 0 || attempt.AlternativesTried > 0 {
		session.State.Summary.Retried++
	}
	if attempt.HintsUsed > 0 {
		session.State.Summary.UsedHint++
	}
	if attempt.Revealed {
		session.State.Summary.Revealed++
	}
	outcome := promptOutcome(attempt)
	completedStepIDs := []string(nil)
	if session.Mode == OpeningModeLesson {
		lesson := course.Lessons[session.LessonID]
		session.StepIndex++
		if session.StepIndex >= len(lesson.Steps) {
			session.Status = OpeningStatusCompleted
			session.State.Position.CurrentFEN = finalFEN
			session.State.Position.PositionID = finalPositionID
			session.State.Attempt = nil
			completedStepIDs = lessonStepIDs(lesson)
		} else {
			session.State, err = s.stateForLessonStep(course, lesson.Steps[session.StepIndex], session.State, s.now().UTC())
			if err != nil {
				return OpeningStepResult{}, err
			}
		}
	} else {
		session.State.Review.Index++
		session.StepIndex = session.State.Review.Index
		if session.State.Review.Index >= len(session.State.Review.PromptIDs) {
			session.Status = OpeningStatusCompleted
			session.State.Position.CurrentFEN = finalFEN
			session.State.Position.PositionID = finalPositionID
			session.State.Attempt = nil
		} else {
			nextPromptID := session.State.Review.PromptIDs[session.State.Review.Index]
			session.State, err = s.stateForReviewPrompt(course, nextPromptID, session.State, s.now().UTC())
			if err != nil {
				return OpeningStepResult{}, err
			}
		}
	}
	if err := s.store.CompletePrompt(ctx, PromptCompletion{
		Session: session, Attempt: attempt,
		SemanticFingerprint: prompt.SemanticFingerprint,
		Outcome:             outcome, CompletedStepIDs: completedStepIDs,
	}, s.now().UTC()); err != nil {
		return OpeningStepResult{}, err
	}
	view, err := s.sessionView(course, session)
	return OpeningStepResult{
		Session: view, StepCompleted: true, Feedback: FeedbackExpected,
		AppliedMoves: frames, FinalFEN: finalFEN,
	}, err
}

func (s *Service) loadLessonStep(
	ctx context.Context,
	sessionID string,
) (StoredSession, CompiledCourse, Lesson, LessonStep, error) {
	session, err := s.store.LoadSession(ctx, sessionID)
	if err != nil {
		return StoredSession{}, CompiledCourse{}, Lesson{}, LessonStep{}, err
	}
	if session.Status != OpeningStatusActive {
		return StoredSession{}, CompiledCourse{}, Lesson{}, LessonStep{}, fmt.Errorf("opening session %q is not active", sessionID)
	}
	if session.Mode != OpeningModeLesson {
		return StoredSession{}, CompiledCourse{}, Lesson{}, LessonStep{}, errors.New("review steps cannot be advanced without a move")
	}
	course, err := s.catalog.LoadGeneration(ctx, session.GenerationID)
	if err != nil {
		return StoredSession{}, CompiledCourse{}, Lesson{}, LessonStep{}, err
	}
	lesson, exists := course.Lessons[session.LessonID]
	if !exists || session.StepIndex < 0 || session.StepIndex >= len(lesson.Steps) {
		return StoredSession{}, CompiledCourse{}, Lesson{}, LessonStep{}, errors.New("opening session lesson step is unavailable")
	}
	return session, course, lesson, lesson.Steps[session.StepIndex], nil
}

func (s *Service) loadPromptStep(
	ctx context.Context,
	sessionID string,
) (StoredSession, CompiledCourse, LessonStep, CompiledPrompt, error) {
	session, err := s.store.LoadSession(ctx, sessionID)
	if err != nil {
		return StoredSession{}, CompiledCourse{}, LessonStep{}, CompiledPrompt{}, err
	}
	if session.Status != OpeningStatusActive {
		return StoredSession{}, CompiledCourse{}, LessonStep{}, CompiledPrompt{}, fmt.Errorf("opening session %q is not active", sessionID)
	}
	course, err := s.catalog.LoadGeneration(ctx, session.GenerationID)
	if err != nil {
		return StoredSession{}, CompiledCourse{}, LessonStep{}, CompiledPrompt{}, err
	}
	var step LessonStep
	if session.Mode == OpeningModeLesson {
		lesson, exists := course.Lessons[session.LessonID]
		if !exists || session.StepIndex < 0 || session.StepIndex >= len(lesson.Steps) {
			return StoredSession{}, CompiledCourse{}, LessonStep{}, CompiledPrompt{}, errors.New("opening session lesson step is unavailable")
		}
		step = lesson.Steps[session.StepIndex]
	} else {
		if session.State.Review == nil || session.State.Review.Index < 0 || session.State.Review.Index >= len(session.State.Review.PromptIDs) {
			return StoredSession{}, CompiledCourse{}, LessonStep{}, CompiledPrompt{}, errors.New("opening review prompt is unavailable")
		}
		step, err = reviewStep(course, session.State.Review.PromptIDs[session.State.Review.Index])
		if err != nil {
			return StoredSession{}, CompiledCourse{}, LessonStep{}, CompiledPrompt{}, err
		}
	}
	if step.Kind != StepTry && step.Kind != StepBranch && step.Kind != StepRecall {
		return StoredSession{}, CompiledCourse{}, LessonStep{}, CompiledPrompt{}, fmt.Errorf("%s opening step does not accept moves", step.Kind)
	}
	prompt, exists := course.Prompts[step.PromptID]
	if !exists {
		return StoredSession{}, CompiledCourse{}, LessonStep{}, CompiledPrompt{}, fmt.Errorf("opening prompt %q is unavailable", step.PromptID)
	}
	if _, err := attemptRecord(session.State.Attempt); err != nil {
		return StoredSession{}, CompiledCourse{}, LessonStep{}, CompiledPrompt{}, err
	}
	return session, course, step, prompt, nil
}

func (s *Service) stateForLessonStep(
	course CompiledCourse,
	step LessonStep,
	previous SessionState,
	now time.Time,
) (SessionState, error) {
	position, exists := course.Positions[step.PositionID]
	if !exists {
		return SessionState{}, fmt.Errorf("opening position %q is unavailable", step.PositionID)
	}
	state := resetAttemptState(previous)
	state.Position.PositionID = step.PositionID
	state.Position.CurrentFEN = position.FEN
	if step.PromptID != "" {
		state.Attempt = &AttemptState{
			PromptID: step.PromptID, AttemptID: nextAttemptID(), StartedAt: now,
		}
	}
	return state, nil
}

func (s *Service) stateForReviewPrompt(
	course CompiledCourse,
	promptID string,
	previous SessionState,
	now time.Time,
) (SessionState, error) {
	prompt, exists := course.Prompts[promptID]
	if !exists {
		return SessionState{}, fmt.Errorf("opening prompt %q is unavailable", promptID)
	}
	position, exists := course.Positions[prompt.PositionID]
	if !exists {
		return SessionState{}, fmt.Errorf("opening position %q is unavailable", prompt.PositionID)
	}
	state := resetAttemptState(previous)
	if state.Review == nil {
		return SessionState{}, errors.New("review session requires a review cursor")
	}
	state.Position.PositionID = prompt.PositionID
	state.Position.CurrentFEN = position.FEN
	state.Attempt = &AttemptState{
		PromptID: promptID, AttemptID: nextAttemptID(), StartedAt: now,
	}
	return state, nil
}

func resetAttemptState(state SessionState) SessionState {
	state.Attempt = nil
	return state
}

func promptOutcome(state AttemptRecord) ReviewOutcome {
	switch {
	case state.Revealed:
		return ReviewRevealed
	case state.HintsUsed > 0:
		return ReviewHinted
	case state.IncorrectMoves > 0:
		return ReviewMissed
	default:
		return ReviewClean
	}
}

func (s *Service) applyMoveIDs(
	course CompiledCourse,
	startFEN string,
	moveIDs []string,
) ([]domain.AppliedMove, string, error) {
	frames := make([]domain.AppliedMove, 0, len(moveIDs))
	currentFEN := startFEN
	for _, moveID := range moveIDs {
		move, exists := course.Moves[moveID]
		if !exists {
			return nil, "", fmt.Errorf("opening move %q is unavailable", moveID)
		}
		nextFEN, err := s.rules.ApplyUCI(currentFEN, move.UCI)
		if err != nil {
			return nil, "", fmt.Errorf("apply opening move %q: %w", move.UCI, err)
		}
		frames = append(frames, domain.AppliedMove{UCI: move.UCI, ResultingFEN: nextFEN})
		currentFEN = nextFEN
	}
	return frames, currentFEN, nil
}

func (s *Service) sessionView(course CompiledCourse, session StoredSession) (OpeningSessionView, error) {
	view := OpeningSessionView{
		SessionID: session.ID, Mode: session.Mode, Status: session.Status,
		CourseID: session.CourseID, GenerationID: session.GenerationID,
		LessonID: session.LessonID, Depth: session.Depth,
	}
	if session.Status == OpeningStatusCompleted {
		view.Summary = openingSummary(session.State)
		return view, nil
	}
	var stepView OpeningStepView
	var err error
	if session.Mode == OpeningModeLesson {
		lesson, exists := course.Lessons[session.LessonID]
		if !exists || session.StepIndex < 0 || session.StepIndex >= len(lesson.Steps) {
			return OpeningSessionView{}, errors.New("opening session lesson step is unavailable")
		}
		stepView, err = s.lessonStepView(course, lesson, session)
	} else {
		stepView, err = s.reviewStepView(course, session)
	}
	if err != nil {
		return OpeningSessionView{}, err
	}
	view.Current = &stepView
	return view, nil
}

func (s *Service) lessonStepView(
	course CompiledCourse,
	lesson Lesson,
	session StoredSession,
) (OpeningStepView, error) {
	step := lesson.Steps[session.StepIndex]
	return s.buildStepView(course, step, session, session.StepIndex+1, len(lesson.Steps))
}

func (s *Service) reviewStepView(
	course CompiledCourse,
	session StoredSession,
) (OpeningStepView, error) {
	if session.State.Review == nil || session.State.Review.Index < 0 || session.State.Review.Index >= len(session.State.Review.PromptIDs) {
		return OpeningStepView{}, errors.New("opening review prompt is unavailable")
	}
	step, err := reviewStep(course, session.State.Review.PromptIDs[session.State.Review.Index])
	if err != nil {
		return OpeningStepView{}, err
	}
	return s.buildStepView(
		course, step, session, session.State.Review.Index+1, len(session.State.Review.PromptIDs),
	)
}

func reviewStep(course CompiledCourse, promptID string) (LessonStep, error) {
	prompt, exists := course.Prompts[promptID]
	if !exists {
		return LessonStep{}, fmt.Errorf("opening prompt %q is unavailable", promptID)
	}
	position := course.Positions[prompt.PositionID]
	return LessonStep{
		StepID: "review-" + promptID, Kind: StepRecall, PositionID: prompt.PositionID,
		Title: "Review " + position.Label, Instruction: "Play the course move from memory.",
		NoteIDs: []string{}, MoveIDs: []string{}, PromptID: promptID,
	}, nil
}

func (s *Service) buildStepView(
	course CompiledCourse,
	step LessonStep,
	session StoredSession,
	number int,
	total int,
) (OpeningStepView, error) {
	position, exists := course.Positions[step.PositionID]
	if !exists {
		return OpeningStepView{}, fmt.Errorf("opening position %q is unavailable", step.PositionID)
	}
	view := OpeningStepView{
		StepID: step.StepID, Kind: step.Kind, Title: step.Title,
		Instruction: step.Instruction, PositionID: step.PositionID,
		CurrentFEN: session.State.Position.CurrentFEN, Orientation: course.Pack.Perspective,
		LegalMoves: []string{}, NoteTexts: s.noteTexts(course, step),
		StepNumber: number, StepTotal: total,
	}
	if session.State.Attempt != nil {
		view.HintLevel = session.State.Attempt.HintLevel
		view.CanReveal = session.State.Attempt.HintLevel >= 4
	}
	if view.CurrentFEN == "" {
		view.CurrentFEN = position.FEN
	}
	if step.PromptID != "" {
		prompt := course.Prompts[step.PromptID]
		primary := course.Moves[prompt.PrimaryMoveID]
		view.VariationName = primary.VariationName
		legal, err := s.rules.LegalMoves(view.CurrentFEN)
		if err != nil {
			return OpeningStepView{}, fmt.Errorf("list opening moves: %w", err)
		}
		view.LegalMoves = legal
	} else if step.Kind == StepWatch && len(step.MoveIDs) > 0 {
		view.VariationName = course.Moves[step.MoveIDs[len(step.MoveIDs)-1]].VariationName
	}
	return view, nil
}

func (s *Service) noteTexts(course CompiledCourse, step LessonStep) []string {
	ids := append([]string{}, step.NoteIDs...)
	if position, exists := course.Positions[step.PositionID]; exists {
		ids = append(ids, position.NoteIDs...)
	}
	if prompt, exists := course.Prompts[step.PromptID]; exists {
		if primary, ok := course.Moves[prompt.PrimaryMoveID]; ok {
			ids = append(ids, primary.NoteIDs...)
		}
	}
	texts := []string{}
	seen := map[string]bool{}
	for _, id := range ids {
		note, exists := course.Notes[id]
		if exists && !seen[note.Text] {
			seen[note.Text] = true
			texts = append(texts, note.Text)
		}
	}
	return texts
}

func (s *Service) planHint(course CompiledCourse, prompt CompiledPrompt) string {
	step := LessonStep{PositionID: prompt.PositionID, PromptID: prompt.PromptID, NoteIDs: []string{}}
	texts := s.noteTexts(course, step)
	if len(texts) > 0 {
		return texts[0]
	}
	return "Remember the plan for this position."
}

func openingSummary(state SessionState) *OpeningSummary {
	return &OpeningSummary{
		TotalPrompts: state.Summary.CompletedPrompts, PositionsRecalled: state.Summary.PositionsRecalled,
		BranchesRecognized: state.Summary.BranchesRecognized, Retried: state.Summary.Retried,
		UsedHint: state.Summary.UsedHint, Revealed: state.Summary.Revealed,
	}
}
