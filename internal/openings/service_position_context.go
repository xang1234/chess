package openings

import "fmt"

func buildPositionContext(
	course CompiledCourse,
	positionID string,
	depth Depth,
	route activityRoute,
) (OpeningPositionContextView, error) {
	position, exists := course.Positions[positionID]
	if !exists {
		return OpeningPositionContextView{}, fmt.Errorf(
			"opening position %q is unavailable", positionID,
		)
	}
	if len(route.MoveIDs) != len(route.Frames) || len(route.MoveIDs) != len(route.SAN) {
		return OpeningPositionContextView{}, fmt.Errorf(
			"opening position %q has a misaligned route", positionID,
		)
	}
	view := OpeningPositionContextView{
		PositionID: positionID,
		Label:      position.Label,
		Evaluation: position.Evaluation,
		PlanText:   firstNoteTextOfKinds(course, position.NoteIDs, "plan"),
		ArrivedVia: []OpeningContextMoveView{},
		NextMoves:  []OpeningContextMoveView{},
	}
	if len(route.MoveIDs) > 0 {
		selectedID := route.MoveIDs[len(route.MoveIDs)-1]
		selected, ok := course.Moves[selectedID]
		if !ok || selected.ToPositionID != positionID {
			return OpeningPositionContextView{}, fmt.Errorf(
				"position %q has invalid route move %q", positionID, selectedID,
			)
		}
		view.ArrivedVia = contextMoves(
			course.VisibleMoves(selected.FromPositionID, depth),
			selectedID,
		)
	}
	view.NextMoves = contextMoves(course.VisibleMoves(positionID, depth), "")
	return view, nil
}

func contextMoves(moves []CompiledMove, selectedID string) []OpeningContextMoveView {
	result := make([]OpeningContextMoveView, 0, len(moves))
	for _, move := range moves {
		result = append(result, OpeningContextMoveView{
			MoveID:        move.MoveID,
			SAN:           move.SAN,
			Role:          move.TrainingRole,
			VariationName: move.VariationName,
			Evaluation:    move.Evaluation,
			Selected:      move.MoveID == selectedID,
		})
	}
	return result
}

func firstNoteTextOfKinds(
	course CompiledCourse,
	noteIDs []string,
	kinds ...string,
) string {
	for _, kind := range kinds {
		for _, noteID := range noteIDs {
			note, exists := course.Notes[noteID]
			if exists && note.Kind == kind {
				return note.Text
			}
		}
	}
	return ""
}
