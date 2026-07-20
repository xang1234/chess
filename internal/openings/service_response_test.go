package openings

import (
	"encoding/json"
	"strings"
	"testing"

	"chess-trainer/internal/domain"
)

func TestOpeningSessionViewJSONUsesStrictDiscriminatingFields(t *testing.T) {
	view := OpeningSessionView{
		SessionID: "session-1", Mode: OpeningModeLesson, Status: OpeningStatusActive,
		CourseID: "italian", GenerationID: "generation-1", LessonID: "lesson-1",
		Depth: DepthReference,
		Current: &OpeningActivityView{
			ActivityID: "activity-1", Kind: ActivityConcept, Title: "The centre", Required: true,
			Instruction: "Learn the plan.", PositionID: "root", CurrentFEN: "fen",
			Orientation: PerspectiveWhite, LegalMoves: []string{}, TeachingNoteTexts: []string{}, ReferenceNoteTexts: []string{},
			Comparison: []OpeningActivityLine{}, Annotations: []BoardAnnotation{}, MovesToHere: []domain.AppliedMove{},
			ActivityNumber: 1, ActivityTotal: 3, RequiredIdeas: 3, ReferenceSections: []OpeningReferenceSection{},
		},
	}
	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, want := range []string{`"mode":"lesson"`, `"status":"active"`, `"legalMoves":[]`, `"teachingNoteTexts":[]`, `"referenceNoteTexts":[]`} {
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
