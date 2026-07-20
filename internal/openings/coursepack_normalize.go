package openings

import (
	"fmt"
	"sort"
)

func NormalizeCoursePack(pack CoursePack) (CoursePack, error) {
	switch pack.SchemaVersion {
	case 1:
		return normalizeLegacyCoursePack(pack)
	case 2:
		for index, lesson := range pack.Lessons {
			if len(lesson.Steps) != 0 {
				return CoursePack{}, fmt.Errorf("schema 2 lesson steps are not allowed at lessons[%d]", index)
			}
		}
		return pack, nil
	default:
		return CoursePack{}, fmt.Errorf("unsupported course schema version %d", pack.SchemaVersion)
	}
}

func normalizeLegacyCoursePack(pack CoursePack) (CoursePack, error) {
	if len(pack.LessonEdges) != 0 {
		return CoursePack{}, fmt.Errorf("schema 1 lesson edges are not allowed")
	}
	for lessonIndex := range pack.Lessons {
		lesson := &pack.Lessons[lessonIndex]
		if len(lesson.Activities) != 0 {
			return CoursePack{}, fmt.Errorf("schema 1 lesson activities are not allowed at lessons[%d]", lessonIndex)
		}
		lesson.Activities = make([]LessonActivity, 0, len(lesson.Steps))
		for _, step := range lesson.Steps {
			kind, err := legacyActivityKind(step.Kind)
			if err != nil {
				return CoursePack{}, fmt.Errorf("lesson %q step %q: %w", lesson.LessonID, step.StepID, err)
			}
			lesson.Activities = append(lesson.Activities, LessonActivity{
				ActivityID:  step.StepID,
				Kind:        kind,
				Title:       step.Title,
				Instruction: step.Instruction,
				Required:    true,
				PositionID:  step.PositionID,
				NoteIDs:     append([]string{}, step.NoteIDs...),
				MoveIDs:     append([]string{}, step.MoveIDs...),
				PromptID:    step.PromptID,
			})
		}
	}
	pack.LessonEdges = legacyLessonEdges(pack)
	return pack, nil
}

func legacyActivityKind(kind StepKind) (ActivityKind, error) {
	switch kind {
	case StepExplain:
		return ActivityConcept, nil
	case StepWatch:
		return ActivityDemonstration, nil
	case StepTry, StepBranch, StepRecall:
		return ActivityDecision, nil
	default:
		return "", fmt.Errorf("unsupported legacy step kind %q", kind)
	}
}

func legacyLessonEdges(pack CoursePack) []LessonEdge {
	chapterOrder := make(map[string]int, len(pack.Chapters))
	for _, chapter := range pack.Chapters {
		chapterOrder[chapter.ChapterID] = chapter.Ordinal
	}
	ordered := append([]Lesson{}, pack.Lessons...)
	sort.SliceStable(ordered, func(left, right int) bool {
		leftChapter := chapterOrder[ordered[left].ChapterID]
		rightChapter := chapterOrder[ordered[right].ChapterID]
		if leftChapter != rightChapter {
			return leftChapter < rightChapter
		}
		if ordered[left].Ordinal != ordered[right].Ordinal {
			return ordered[left].Ordinal < ordered[right].Ordinal
		}
		return ordered[left].LessonID < ordered[right].LessonID
	})
	edges := make([]LessonEdge, 0, max(0, len(ordered)-1))
	siblingOrdinals := map[string]int{}
	for index := 1; index < len(ordered); index++ {
		parentIndex := index - 1
		childRank, childDepthOK := depthRank(ordered[index].MinimumDepth)
		if childDepthOK {
			for parentIndex > 0 {
				parentRank, parentDepthOK := depthRank(ordered[parentIndex].MinimumDepth)
				if parentDepthOK && parentRank <= childRank {
					break
				}
				parentIndex--
			}
		}
		parentID := ordered[parentIndex].LessonID
		siblingOrdinals[parentID]++
		edges = append(edges, LessonEdge{
			EdgeID:       fmt.Sprintf("legacy-edge-%04d", index),
			FromLessonID: parentID,
			ToLessonID:   ordered[index].LessonID,
			Ordinal:      siblingOrdinals[parentID],
			Kind:         EdgeContinuation,
			MinimumDepth: ordered[index].MinimumDepth,
		})
	}
	return edges
}
