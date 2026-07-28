package openings

import "fmt"

func (s *Service) moveFeedbackDetail(
	course CompiledCourse,
	state SessionState,
	depth Depth,
	prompt CompiledPrompt,
	uci string,
) (OpeningMoveFeedbackDetail, error) {
	primary, exists := course.Moves[prompt.PrimaryMoveID]
	if !exists {
		return OpeningMoveFeedbackDetail{}, fmt.Errorf(
			"opening prompt %q has no primary move", prompt.PromptID,
		)
	}
	position, exists := course.Positions[state.Position.PositionID]
	if !exists {
		return OpeningMoveFeedbackDetail{}, fmt.Errorf(
			"opening position %q is unavailable", state.Position.PositionID,
		)
	}
	playedSAN, err := s.rules.SAN(state.Position.CurrentFEN, uci)
	if err != nil {
		return OpeningMoveFeedbackDetail{}, fmt.Errorf(
			"write opening SAN for %q: %w", uci, err,
		)
	}
	detail := OpeningMoveFeedbackDetail{
		Coverage:    FeedbackCoverageUnknown,
		PlayedSAN:   playedSAN,
		Evaluation:  Evaluation{Code: EvaluationNone},
		ExpectedSAN: primary.SAN,
		PlanText:    firstNoteTextOfKinds(course, position.NoteIDs, "plan"),
	}
	for _, moveID := range course.Outgoing[state.Position.PositionID] {
		move, ok := course.Moves[moveID]
		if !ok || move.UCI != uci {
			continue
		}
		detail.PlayedSAN = move.SAN
		detail.VariationName = move.VariationName
		detail.Evaluation = move.Evaluation
		detail.NoteText = firstNoteTextOfKinds(
			course,
			move.NoteIDs,
			"explanation",
			"plan",
			"warning",
			"evaluation",
		)
		if visibleAtDepth(move.MinimumDepth, depth) {
			detail.Coverage = FeedbackCoverageVisible
		} else {
			detail.Coverage = FeedbackCoverageDeeper
			detail.RequiredDepth = move.MinimumDepth
		}
		return detail, nil
	}
	return detail, nil
}

func feedbackFallbackMessage(detail OpeningMoveFeedbackDetail) string {
	switch detail.Coverage {
	case FeedbackCoverageVisible:
		return fmt.Sprintf(
			"That is a known course move, but this lesson is practicing %s.",
			detail.ExpectedSAN,
		)
	case FeedbackCoverageDeeper:
		return fmt.Sprintf(
			"That move is covered at %s depth. This lesson is practicing %s.",
			detail.RequiredDepth,
			detail.ExpectedSAN,
		)
	default:
		return fmt.Sprintf(
			"That move is playable, but this lesson is practicing %s.",
			detail.ExpectedSAN,
		)
	}
}
