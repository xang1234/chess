package openings

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestBuildStepViewSeparatesTeachingNotesFromReferenceNotes(t *testing.T) {
	course := CompiledCourse{
		Pack: CoursePack{Perspective: PerspectiveWhite},
		Positions: map[string]CompiledPosition{
			"position": {
				ID:      "position",
				FEN:     "reference-fen",
				NoteIDs: []string{"reference-note", "teaching-copy"},
			},
		},
		Notes: map[string]Note{
			"teaching-note":  {NoteID: "teaching-note", Text: "Keep this teaching note visible."},
			"teaching-copy":  {NoteID: "teaching-copy", Text: "Keep this teaching note visible."},
			"reference-note": {NoteID: "reference-note", Text: "Keep this detailed note in the reference section."},
		},
	}
	step := LessonStep{
		StepID: "explain", Kind: StepExplain, PositionID: "position",
		Title: "Plan", Instruction: "Learn the plan.", NoteIDs: []string{"teaching-note"},
	}
	session := StoredSession{State: SessionState{Position: PositionState{CurrentFEN: "current-fen"}}}

	view, err := (&Service{}).buildStepView(course, step, session, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Keep this teaching note visible."}; !reflect.DeepEqual(view.NoteTexts, want) {
		t.Fatalf("teaching notes = %v, want %v", view.NoteTexts, want)
	}

	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		ReferenceNoteTexts []string `json:"referenceNoteTexts"`
	}
	if err := json.Unmarshal(encoded, &contract); err != nil {
		t.Fatal(err)
	}
	if want := []string{"Keep this detailed note in the reference section."}; !reflect.DeepEqual(contract.ReferenceNoteTexts, want) {
		t.Fatalf("reference notes = %v, want %v", contract.ReferenceNoteTexts, want)
	}
}
