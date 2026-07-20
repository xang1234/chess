package openings

import (
	"context"
	"errors"
	"fmt"
	"time"

	"chess-trainer/internal/domain"
)

// Advance is the one-release compatibility name for advancing a passive activity.
func (s *Service) Advance(ctx context.Context, sessionID string) (OpeningStepResult, error) {
	return s.AdvanceActivity(ctx, sessionID)
}

func (s *Service) stateForActivity(
	course CompiledCourse,
	activity LessonActivity,
	previous SessionState,
	now time.Time,
) (SessionState, error) {
	state := resetAttemptState(previous)
	if activity.PositionID != "" {
		position, exists := course.Positions[activity.PositionID]
		if !exists {
			return SessionState{}, fmt.Errorf("opening position %q is unavailable", activity.PositionID)
		}
		state.Position.PositionID = activity.PositionID
		state.Position.CurrentFEN = position.FEN
	}
	if state.Position.PositionID == "" || state.Position.CurrentFEN == "" {
		return SessionState{}, fmt.Errorf("opening activity %q has no available position", activity.ActivityID)
	}
	if activity.Kind == ActivityDecision {
		state.Attempt = &AttemptState{
			PromptID: activity.PromptID, AttemptID: nextAttemptID(), StartedAt: now,
		}
	}
	return state, nil
}

// stateForLessonStep remains until legacy generation rebase is migrated to stable activity IDs.
func (s *Service) stateForLessonStep(
	course CompiledCourse,
	step LessonStep,
	previous SessionState,
	now time.Time,
) (SessionState, error) {
	kind, err := legacyActivityKind(step.Kind)
	if err != nil {
		return SessionState{}, err
	}
	return s.stateForActivity(course, LessonActivity{
		ActivityID: step.StepID, Kind: kind, Title: step.Title, Instruction: step.Instruction,
		Required: true, PositionID: step.PositionID, NoteIDs: step.NoteIDs,
		MoveIDs: step.MoveIDs, PromptID: step.PromptID,
	}, previous, now)
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

func noteTextsForIDs(course CompiledCourse, ids []string, excluded map[string]bool) []string {
	texts := []string{}
	seen := map[string]bool{}
	for _, id := range ids {
		note, exists := course.Notes[id]
		if exists && !excluded[note.Text] && !seen[note.Text] {
			seen[note.Text] = true
			texts = append(texts, note.Text)
		}
	}
	return texts
}

func (s *Service) activityReferenceNoteTexts(
	course CompiledCourse,
	activity LessonActivity,
	teachingNotes []string,
) []string {
	ids := []string{}
	if position, exists := course.Positions[activity.PositionID]; exists {
		ids = append(ids, position.NoteIDs...)
	}
	if prompt, exists := course.Prompts[activity.PromptID]; exists {
		if primary, ok := course.Moves[prompt.PrimaryMoveID]; ok {
			ids = append(ids, primary.NoteIDs...)
		}
	}
	excluded := make(map[string]bool, len(teachingNotes))
	for _, text := range teachingNotes {
		excluded[text] = true
	}
	return noteTextsForIDs(course, ids, excluded)
}

func (s *Service) planHint(course CompiledCourse, prompt CompiledPrompt) string {
	teaching := []string{}
	reference := s.activityReferenceNoteTexts(course, LessonActivity{
		PositionID: prompt.PositionID, PromptID: prompt.PromptID,
	}, teaching)
	if len(reference) > 0 {
		return reference[0]
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
