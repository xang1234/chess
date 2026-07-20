package openings

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOpeningSessionViewJSONUsesStrictDiscriminatingFields(t *testing.T) {
	view := OpeningSessionView{
		SessionID: "session-1", Mode: OpeningModeLesson, Status: OpeningStatusActive,
		CourseID: "italian", GenerationID: "generation-1", LessonID: "lesson-1",
		Depth: DepthReference,
		Current: &OpeningStepView{
			StepID: "step-1", Kind: StepExplain, Title: "The centre",
			Instruction: "Learn the plan.", PositionID: "root", CurrentFEN: "fen",
			Orientation: PerspectiveWhite, LegalMoves: []string{}, NoteTexts: []string{}, ReferenceNoteTexts: []string{},
			StepNumber: 1, StepTotal: 5,
		},
	}
	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, want := range []string{`"mode":"lesson"`, `"status":"active"`, `"legalMoves":[]`, `"noteTexts":[]`, `"referenceNoteTexts":[]`} {
		if !strings.Contains(text, want) {
			t.Fatalf("json = %s, want %s", text, want)
		}
	}
	if strings.Contains(text, `"summary"`) || strings.Contains(text, `"notice"`) {
		t.Fatalf("json = %s, unexpected optional field", text)
	}
}

func TestOpeningStepResultOmitsEmptyMoveFeedback(t *testing.T) {
	encoded, err := json.Marshal(OpeningStepResult{Session: OpeningSessionView{}})
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if strings.Contains(text, `"feedback"`) || strings.Contains(text, `"appliedMoves"`) {
		t.Fatalf("json = %s", text)
	}
}
