package openings

import (
	"errors"
	"reflect"
	"testing"

	"chess-trainer/internal/chessrules"
)

func TestRequiredActivityIDsExcludesOptionalReference(t *testing.T) {
	pack := decodeTreePack(t)
	pack.Lessons[0].Activities = append(pack.Lessons[0].Activities, LessonActivity{
		ActivityID: "giuoco-reference", Kind: ActivityReference, Title: "Deeper analysis",
		Instruction: "Read the source details when useful.", Required: false,
		PositionID: "after-c3", NoteIDs: []string{"italian-overview"}, MoveIDs: []string{},
	})
	compiled, err := Compile(pack, chessrules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"giuoco-concept", "giuoco-c3-decision", "giuoco-recap"}
	if got := RequiredActivityIDs(compiled.Lessons["giuoco-plan"]); !reflect.DeepEqual(got, want) {
		t.Fatalf("required IDs = %v, want %v", got, want)
	}
}

func TestCompileActivitiesRejectsKindSpecificErrors(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		mutate func(*CoursePack)
	}{
		{
			name: "required reference", code: "reference_required",
			mutate: func(pack *CoursePack) {
				pack.Lessons[0].Activities = append(pack.Lessons[0].Activities, LessonActivity{
					ActivityID: "required-reference", Kind: ActivityReference, Title: "Analysis",
					Instruction: "Read a detailed line.", Required: true, PositionID: "after-c3",
					NoteIDs: []string{}, MoveIDs: []string{},
				})
			},
		},
		{
			name: "short comparison", code: "comparison_lines",
			mutate: func(pack *CoursePack) {
				pack.Lessons[0].Activities = append(pack.Lessons[0].Activities, LessonActivity{
					ActivityID: "comparison", Kind: ActivityComparison, Title: "Compare plans",
					Instruction: "Contrast the plans.", Required: true, PositionID: "after-bc5",
					NoteIDs: []string{}, MoveIDs: []string{},
					Comparison: []ActivityLine{{Label: "Only line", MoveIDs: []string{"white-c3"}}},
				})
			},
		},
		{
			name: "disconnected demonstration", code: "disconnected_demonstration_path",
			mutate: func(pack *CoursePack) {
				pack.Lessons[0].Activities = append(pack.Lessons[0].Activities, LessonActivity{
					ActivityID: "demonstration", Kind: ActivityDemonstration, Title: "Watch",
					Instruction: "Watch the moves.", Required: true, PositionID: "after-bc5",
					NoteIDs: []string{}, MoveIDs: []string{"white-e4"},
				})
			},
		},
		{
			name: "arrow without destination", code: "invalid_annotation",
			mutate: func(pack *CoursePack) {
				pack.Lessons[0].Activities[0].Annotations = []BoardAnnotation{{Kind: "arrow", From: "c2"}}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pack := decodeTreePack(t)
			test.mutate(&pack)
			_, err := Compile(pack, chessrules.Rules{})
			var validation *ValidationError
			if !errors.As(err, &validation) || !hasDiagnosticCode(validation.Diagnostics, test.code) {
				t.Fatalf("error=%v diagnostics=%+v want=%s", err, validation, test.code)
			}
		})
	}
}

func TestCompileLegacyActivitiesRemainExemptFromDuplicateDecisionRule(t *testing.T) {
	if _, err := Compile(decodeMiniPack(t), chessrules.Rules{}); err != nil {
		t.Fatal(err)
	}
}
