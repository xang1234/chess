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
		CourseMap: &OpeningCourseMapView{
			CompletedIdeas: 1, TotalIdeas: 3,
			Entries: []OpeningCourseMapEntryView{{
				LessonID: "lesson-1", MoveLabel: "3...Bc5", Title: "The centre",
				State: CourseMapCurrent, Selectable: true,
			}},
		},
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
	for _, want := range []string{`"mode":"lesson"`, `"status":"active"`, `"courseMap":`, `"moveLabel":"3...Bc5"`, `"legalMoves":[]`, `"teachingNoteTexts":[]`, `"referenceNoteTexts":[]`} {
		if !strings.Contains(text, want) {
			t.Fatalf("json = %s, want %s", text, want)
		}
	}
	if strings.Contains(text, `"summary"`) || strings.Contains(text, `"notice"`) {
		t.Fatalf("json = %s, unexpected optional field", text)
	}
	for _, other := range []OpeningSessionView{
		{Mode: OpeningModeReview, Status: OpeningStatusActive},
		{Mode: OpeningModeLesson, Status: OpeningStatusCompleted},
	} {
		encoded, err = json.Marshal(other)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), `"courseMap"`) {
			t.Fatalf("non-active-lesson json = %s, unexpected course map", encoded)
		}
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
