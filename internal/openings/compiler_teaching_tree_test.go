package openings

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"testing"

	"chess-trainer/internal/chessrules"
)

func decodeTreePack(t *testing.T) CoursePack {
	t.Helper()
	contents, err := os.ReadFile("testdata/tree.ctcourse")
	if err != nil {
		t.Fatal(err)
	}
	pack, err := DecodeCoursePack(bytes.NewReader(contents))
	if err != nil {
		t.Fatal(err)
	}
	return pack
}

func compileTreeCourse(t *testing.T) CompiledCourse {
	t.Helper()
	compiled, err := Compile(decodeTreePack(t), chessrules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	return compiled
}

func TestCompileTeachingTreeBuildsDeterministicIndexes(t *testing.T) {
	compiled := compileTreeCourse(t)
	index := buildTeachingTreeIndex(compiled)
	if compiled.RootLessonID != "giuoco-plan" {
		t.Fatalf("root = %q", compiled.RootLessonID)
	}
	wantChildren := []LessonEdge{compiled.Pack.LessonEdges[0]}
	if !reflect.DeepEqual(compiled.LessonChildren["giuoco-plan"], wantChildren) {
		t.Fatalf("children = %#v, want %#v", compiled.LessonChildren["giuoco-plan"], wantChildren)
	}
	if parent := compiled.LessonParent["two-knights-plan"]; parent.EdgeID != "giuoco-to-two-knights" {
		t.Fatalf("parent = %#v", parent)
	}
	if !reflect.DeepEqual(index.children, compiled.LessonChildren) ||
		!reflect.DeepEqual(index.parents, compiled.LessonParent) ||
		!reflect.DeepEqual(index.roots, []string{"giuoco-plan"}) {
		t.Fatalf("index=%+v compiled children=%+v parents=%+v", index, compiled.LessonChildren, compiled.LessonParent)
	}
}

func TestCompileTeachingTreeRejectsStructuralAndActivityErrors(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		mutate func(*CoursePack)
	}{
		{
			name: "cycle", code: "lesson_tree_cycle",
			mutate: func(pack *CoursePack) {
				pack.LessonEdges = append(pack.LessonEdges, LessonEdge{
					EdgeID: "cycle", FromLessonID: "two-knights-plan", ToLessonID: "giuoco-plan",
					Ordinal: 1, Kind: EdgeContinuation, MinimumDepth: DepthStandard,
				})
			},
		},
		{
			name: "second parent", code: "lesson_multiple_parents",
			mutate: func(pack *CoursePack) {
				pack.LessonEdges = append(pack.LessonEdges, LessonEdge{
					EdgeID: "parent-2", FromLessonID: "giuoco-plan", ToLessonID: "two-knights-plan",
					Ordinal: 2, Kind: EdgeAlternative, MinimumDepth: DepthStandard,
				})
			},
		},
		{
			name: "missing prompt", code: "missing_prompt",
			mutate: func(pack *CoursePack) { pack.Lessons[0].Activities[1].PromptID = "" },
		},
		{
			name: "duplicate answer", code: "duplicate_lesson_decision",
			mutate: func(pack *CoursePack) {
				activity := pack.Lessons[0].Activities[1]
				activity.ActivityID = "duplicate-c3"
				pack.Lessons[0].Activities = append(pack.Lessons[0].Activities, activity)
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

func TestCompileTeachingTreeRejectsDisconnectedDepthRoute(t *testing.T) {
	pack := decodeTreePack(t)
	pack.LessonEdges[0].MinimumDepth = DepthReference
	_, err := Compile(pack, chessrules.Rules{})
	var validation *ValidationError
	if !errors.As(err, &validation) || !hasDiagnosticCode(validation.Diagnostics, "lesson_depth_route") {
		t.Fatalf("error=%v diagnostics=%+v", err, validation)
	}
}
