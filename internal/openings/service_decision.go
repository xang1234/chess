package openings

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"chess-trainer/internal/domain"
)

func (s *Service) PlayMove(
	ctx context.Context,
	sessionID string,
	uci string,
) (OpeningActivityResult, error) {
	session, course, lesson, activity, prompt, err := s.loadDecision(ctx, sessionID)
	if err != nil {
		return OpeningActivityResult{}, err
	}
	uci = strings.TrimSpace(uci)
	if uci == "" {
		return OpeningActivityResult{}, errors.New("opening move is required")
	}
	if _, err := s.rules.ApplyUCI(session.State.Position.CurrentFEN, uci); err != nil {
		return OpeningActivityResult{}, fmt.Errorf("illegal opening move %q: %w", uci, err)
	}
	primary := course.Moves[prompt.PrimaryMoveID]
	if uci == primary.UCI {
		return s.completeDecision(ctx, course, session, lesson, activity, prompt)
	}
	for _, moveID := range prompt.AcceptedAlternativeMoveIDs {
		move, exists := course.Moves[moveID]
		if exists && visibleAtDepth(move.MinimumDepth, session.Depth) && move.UCI == uci {
			session.State.Attempt.AlternativesTried++
			if err := s.store.SaveSession(ctx, session, s.now().UTC()); err != nil {
				return OpeningActivityResult{}, err
			}
			view, err := s.sessionView(ctx, course, session)
			return OpeningActivityResult{
				Session: view, Feedback: FeedbackAlternative,
				Message: "That is a playable course alternative. Return to the lesson position and try the course move.",
			}, err
		}
	}
	session.State.Attempt.IncorrectMoves++
	if err := s.store.SaveSession(ctx, session, s.now().UTC()); err != nil {
		return OpeningActivityResult{}, err
	}
	view, err := s.sessionView(ctx, course, session)
	return OpeningActivityResult{
		Session: view, Feedback: FeedbackOffCourse,
		Message: fmt.Sprintf("That move is playable, but this lesson is practicing %s.", primary.SAN),
	}, err
}

func (s *Service) UseHint(ctx context.Context, sessionID string) (OpeningHintResult, error) {
	session, course, _, _, prompt, err := s.loadDecision(ctx, sessionID)
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
	result.Session, err = s.sessionView(ctx, course, session)
	return result, err
}

func (s *Service) Reveal(ctx context.Context, sessionID string) (OpeningActivityResult, error) {
	session, course, lesson, activity, prompt, err := s.loadDecision(ctx, sessionID)
	if err != nil {
		return OpeningActivityResult{}, err
	}
	if session.State.Attempt.HintLevel < 4 {
		return OpeningActivityResult{}, errors.New("use all opening hints before revealing the course move")
	}
	session.State.Attempt.Revealed = true
	return s.completeDecision(ctx, course, session, lesson, activity, prompt)
}

func (s *Service) completeDecision(
	ctx context.Context,
	course CompiledCourse,
	session StoredSession,
	lesson Lesson,
	activity LessonActivity,
	prompt CompiledPrompt,
) (OpeningActivityResult, error) {
	primary := course.Moves[prompt.PrimaryMoveID]
	moveIDs := append([]string{primary.MoveID}, activity.MoveIDs...)
	frames, finalFEN, err := s.applyMoveIDs(course, session.State.Position.CurrentFEN, moveIDs)
	if err != nil {
		return OpeningActivityResult{}, err
	}
	attempt, err := attemptRecord(session.State.Attempt)
	if err != nil {
		return OpeningActivityResult{}, err
	}
	session.State.Position.PlayedMoveIDs = append(session.State.Position.PlayedMoveIDs, moveIDs...)
	updateDecisionSummary(&session.State, attempt)
	outcome := promptOutcome(attempt)
	if session.Mode == OpeningModeLesson {
		return s.completeLessonActivity(
			ctx, course, session, lesson, activity, &attempt,
			prompt.SemanticFingerprint, outcome, frames, finalFEN,
		)
	}
	return s.completeReviewDecision(ctx, course, session, attempt, prompt, frames, finalFEN, moveIDs)
}

func (s *Service) completeReviewDecision(
	ctx context.Context,
	course CompiledCourse,
	session StoredSession,
	attempt AttemptRecord,
	prompt CompiledPrompt,
	frames []domain.AppliedMove,
	finalFEN string,
	moveIDs []string,
) (OpeningActivityResult, error) {
	session.State.Review.Index++
	session.ActivityIndex = session.State.Review.Index
	if session.State.Review.Index >= len(session.State.Review.PromptIDs) {
		session.Status = OpeningStatusCompleted
		session.State.Position.CurrentFEN = finalFEN
		session.State.Position.PositionID = course.Moves[moveIDs[len(moveIDs)-1]].ToPositionID
		session.State.Attempt = nil
	} else {
		nextPromptID := session.State.Review.PromptIDs[session.State.Review.Index]
		var err error
		session.State, err = s.stateForReviewPrompt(course, nextPromptID, session.State, s.now().UTC())
		if err != nil {
			return OpeningActivityResult{}, err
		}
	}
	if err := s.store.CompletePrompt(ctx, PromptCompletion{
		Session: session, Attempt: attempt,
		SemanticFingerprint: prompt.SemanticFingerprint, Outcome: promptOutcome(attempt),
	}, s.now().UTC()); err != nil {
		return OpeningActivityResult{}, err
	}
	view, err := s.sessionView(ctx, course, session)
	return OpeningActivityResult{
		Session: view, ActivityCompleted: true, StepCompleted: true,
		Feedback: FeedbackExpected, AppliedMoves: frames, FinalFEN: finalFEN,
	}, err
}

func updateDecisionSummary(state *SessionState, attempt AttemptRecord) {
	state.Summary.CompletedPrompts++
	state.Summary.PositionsRecalled++
	if attempt.IncorrectMoves > 0 || attempt.AlternativesTried > 0 {
		state.Summary.Retried++
	}
	if attempt.HintsUsed > 0 {
		state.Summary.UsedHint++
	}
	if attempt.Revealed {
		state.Summary.Revealed++
	}
}

func (s *Service) loadDecision(
	ctx context.Context,
	sessionID string,
) (StoredSession, CompiledCourse, Lesson, LessonActivity, CompiledPrompt, error) {
	session, err := s.store.LoadSession(ctx, sessionID)
	if err != nil {
		return StoredSession{}, CompiledCourse{}, Lesson{}, LessonActivity{}, CompiledPrompt{}, err
	}
	if session.Status != OpeningStatusActive {
		return StoredSession{}, CompiledCourse{}, Lesson{}, LessonActivity{}, CompiledPrompt{}, fmt.Errorf("opening session %q is not active", sessionID)
	}
	course, err := s.catalog.LoadGeneration(ctx, session.GenerationID)
	if err != nil {
		return StoredSession{}, CompiledCourse{}, Lesson{}, LessonActivity{}, CompiledPrompt{}, err
	}
	var lesson Lesson
	var activity LessonActivity
	if session.Mode == OpeningModeLesson {
		var exists bool
		lesson, exists = course.Lessons[session.LessonID]
		if !exists || session.ActivityIndex < 0 || session.ActivityIndex >= len(lesson.Activities) {
			return StoredSession{}, CompiledCourse{}, Lesson{}, LessonActivity{}, CompiledPrompt{}, errors.New("opening session lesson activity is unavailable")
		}
		activity = lesson.Activities[session.ActivityIndex]
	} else {
		if session.State.Review == nil || session.State.Review.Index < 0 || session.State.Review.Index >= len(session.State.Review.PromptIDs) {
			return StoredSession{}, CompiledCourse{}, Lesson{}, LessonActivity{}, CompiledPrompt{}, errors.New("opening review prompt is unavailable")
		}
		activity, err = reviewActivity(course, session.State.Review.PromptIDs[session.State.Review.Index])
		if err != nil {
			return StoredSession{}, CompiledCourse{}, Lesson{}, LessonActivity{}, CompiledPrompt{}, err
		}
	}
	if activity.Kind != ActivityDecision {
		return StoredSession{}, CompiledCourse{}, Lesson{}, LessonActivity{}, CompiledPrompt{}, fmt.Errorf("%s opening activity does not accept moves", activity.Kind)
	}
	prompt, exists := course.Prompts[activity.PromptID]
	if !exists {
		return StoredSession{}, CompiledCourse{}, Lesson{}, LessonActivity{}, CompiledPrompt{}, fmt.Errorf("opening prompt %q is unavailable", activity.PromptID)
	}
	if _, err := attemptRecord(session.State.Attempt); err != nil {
		return StoredSession{}, CompiledCourse{}, Lesson{}, LessonActivity{}, CompiledPrompt{}, err
	}
	return session, course, lesson, activity, prompt, nil
}

func reviewActivity(course CompiledCourse, promptID string) (LessonActivity, error) {
	prompt, exists := course.Prompts[promptID]
	if !exists {
		return LessonActivity{}, fmt.Errorf("opening prompt %q is unavailable", promptID)
	}
	position := course.Positions[prompt.PositionID]
	return LessonActivity{
		ActivityID: "review-" + promptID, Kind: ActivityDecision, Required: true,
		PositionID: prompt.PositionID, Title: "Review " + position.Label,
		Instruction: "Play the course move from memory.", NoteIDs: []string{},
		MoveIDs: []string{}, PromptID: promptID,
	}, nil
}

func (s *Service) reviewActivityView(course CompiledCourse, session StoredSession) (OpeningActivityView, error) {
	if session.State.Review == nil || session.State.Review.Index < 0 || session.State.Review.Index >= len(session.State.Review.PromptIDs) {
		return OpeningActivityView{}, errors.New("opening review prompt is unavailable")
	}
	activity, err := reviewActivity(course, session.State.Review.PromptIDs[session.State.Review.Index])
	if err != nil {
		return OpeningActivityView{}, err
	}
	return s.buildActivityView(
		course, Lesson{Activities: []LessonActivity{activity}}, activity, session,
		session.State.Review.Index+1, len(session.State.Review.PromptIDs), session.State.Review.Index,
	)
}
