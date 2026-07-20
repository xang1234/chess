package openings

import (
	"strings"
	"testing"
)

func TestNormalizeCoursePackKeepsLegacyActivityIDs(t *testing.T) {
	pack, err := NormalizeCoursePack(decodeMiniPack(t))
	if err != nil {
		t.Fatal(err)
	}
	lesson := pack.Lessons[0]
	if len(lesson.Activities) != 5 || lesson.Activities[0].ActivityID != lesson.Steps[0].StepID {
		t.Fatalf("activities = %#v", lesson.Activities)
	}
	if lesson.Activities[0].Kind != ActivityConcept || lesson.Activities[1].Kind != ActivityDemonstration || lesson.Activities[2].Kind != ActivityDecision {
		t.Fatalf("kinds = %#v", lesson.Activities)
	}
	if lesson.Activities[2].PromptID != lesson.Steps[2].PromptID || !lesson.Activities[2].Required {
		t.Fatalf("decision = %#v", lesson.Activities[2])
	}
}

func TestNormalizeCoursePackSynthesizesAuthoredLegacyLessonOrder(t *testing.T) {
	pack := decodeMiniPack(t)
	second := pack.Lessons[0]
	second.LessonID = "second-lesson"
	second.Ordinal = 2
	for index := range second.Steps {
		second.Steps[index].StepID = "second-" + second.Steps[index].StepID
	}
	pack.Lessons = append(pack.Lessons, second)

	normalized, err := NormalizeCoursePack(pack)
	if err != nil {
		t.Fatal(err)
	}
	if len(normalized.LessonEdges) != 1 {
		t.Fatalf("edges = %#v", normalized.LessonEdges)
	}
	edge := normalized.LessonEdges[0]
	if edge.FromLessonID != pack.Lessons[0].LessonID || edge.ToLessonID != second.LessonID || edge.Kind != EdgeContinuation {
		t.Fatalf("edge = %#v", edge)
	}
}

func TestNormalizeCoursePackKeepsShallowLegacyLessonsOutOfDeeperBranches(t *testing.T) {
	pack := decodeMiniPack(t)
	root := pack.Lessons[0]
	root.LessonID = "root-quick"
	root.Ordinal = 1
	standard := pack.Lessons[0]
	standard.LessonID = "standard-branch"
	standard.Ordinal = 2
	standard.MinimumDepth = DepthStandard
	quickSibling := pack.Lessons[0]
	quickSibling.LessonID = "quick-sibling"
	quickSibling.Ordinal = 3
	pack.Lessons = []Lesson{root, standard, quickSibling}

	normalized, err := NormalizeCoursePack(pack)
	if err != nil {
		t.Fatal(err)
	}
	if len(normalized.LessonEdges) != 2 {
		t.Fatalf("edges = %#v", normalized.LessonEdges)
	}
	if got := normalized.LessonEdges[1]; got.FromLessonID != "root-quick" || got.ToLessonID != "quick-sibling" || got.Ordinal != 2 {
		t.Fatalf("quick sibling edge = %#v", got)
	}
}

func TestNormalizeCoursePackRejectsVersionSpecificTeachingFields(t *testing.T) {
	tests := []struct {
		name string
		pack CoursePack
		want string
	}{
		{
			name: "schema one activities",
			pack: func() CoursePack {
				pack := decodeMiniPack(t)
				pack.Lessons[0].Activities = []LessonActivity{{ActivityID: "activity", Kind: ActivityConcept}}
				return pack
			}(),
			want: "schema 1 lesson activities",
		},
		{
			name: "schema two steps",
			pack: func() CoursePack {
				pack := decodeMiniPack(t)
				pack.SchemaVersion = 2
				pack.Lessons[0].Activities = []LessonActivity{{ActivityID: "activity", Kind: ActivityConcept}}
				return pack
			}(),
			want: "schema 2 lesson steps",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NormalizeCoursePack(test.pack)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}
