package openings

import (
	"context"
	"fmt"
)

type teachingTreeProjection struct {
	tree                   OpeningTeachingTreeView
	progressByLesson       map[string]LessonProgress
	currentPath            []OpeningPathItem
	recommendedLessonID    string
	recommendedLessonTitle string
	completedLessons       int
	totalLessons           int
	dueReviews             int
}

func (s *Service) projectTeachingTree(
	ctx context.Context,
	course CompiledCourse,
	depth Depth,
	journey CourseJourney,
	resumable *StoredSession,
) (teachingTreeProjection, error) {
	projection := teachingTreeProjection{
		tree: OpeningTeachingTreeView{
			RootLessonID: course.RootLessonID,
			Nodes:        []OpeningTeachingNodeView{},
			Edges:        []OpeningTeachingEdgeView{},
		},
		progressByLesson: map[string]LessonProgress{},
		currentPath:      openingPathItems(course, journey.PathLessonIDs),
	}

	orderedLessonIDs := teachingTreeLessonOrder(course)
	nodeIndexes := make(map[string]int, len(orderedLessonIDs))
	for _, lessonID := range orderedLessonIDs {
		lesson, exists := course.Lessons[lessonID]
		if !exists {
			continue
		}
		requiredIDs := RequiredActivityIDs(lesson)
		progress, err := s.store.LessonProgress(ctx, course.Pack.CourseID, lessonID, requiredIDs)
		if err != nil {
			return teachingTreeProjection{}, fmt.Errorf("load progress for lesson %q: %w", lessonID, err)
		}
		projection.progressByLesson[lessonID] = progress
		visible := visibleAtDepth(lesson.MinimumDepth, depth)
		state := NodeAvailable
		switch {
		case progress.Completed:
			state = NodeCompleted
		case progress.CompletedActivities > 0 || journey.CurrentLessonID == lessonID ||
			(resumable != nil && resumable.CourseID == course.Pack.CourseID && resumable.Mode == OpeningModeLesson && resumable.LessonID == lessonID):
			state = NodeInProgress
		}
		objective := ""
		if len(lesson.Objectives) != 0 {
			objective = lesson.Objectives[0]
		}
		nodeIndexes[lessonID] = len(projection.tree.Nodes)
		projection.tree.Nodes = append(projection.tree.Nodes, OpeningTeachingNodeView{
			LessonID: lesson.LessonID, ChapterID: lesson.ChapterID, Title: lesson.Title,
			Objective: objective, MinimumDepth: lesson.MinimumDepth, Progress: state,
			CompletedActivities: progress.CompletedActivities, RequiredActivities: len(requiredIDs),
			Visible: visible,
		})
		if visible {
			projection.totalLessons++
			if progress.Completed {
				projection.completedLessons++
			}
		}
	}

	for _, edge := range course.Pack.LessonEdges {
		projection.tree.Edges = append(projection.tree.Edges, OpeningTeachingEdgeView{
			EdgeID: edge.EdgeID, FromLessonID: edge.FromLessonID, ToLessonID: edge.ToLessonID,
			Ordinal: edge.Ordinal, Kind: edge.Kind, Label: edge.Label, MinimumDepth: edge.MinimumDepth,
		})
	}

	due, err := s.store.DueReviews(ctx, course.Pack.CourseID, s.now().UTC(), 10000)
	if err != nil {
		return teachingTreeProjection{}, fmt.Errorf("load due opening reviews: %w", err)
	}
	promptLessons := lessonIDsByPrompt(course)
	for _, review := range due {
		prompt, exists := course.Prompts[review.PromptID]
		if !exists || prompt.SemanticFingerprint != review.SemanticFingerprint {
			continue
		}
		visibleReview := false
		for _, lessonID := range promptLessons[review.PromptID] {
			if nodeIndex, exists := nodeIndexes[lessonID]; exists {
				projection.tree.Nodes[nodeIndex].ReviewDue = true
				visibleReview = visibleReview || projection.tree.Nodes[nodeIndex].Visible
			}
		}
		if visibleReview && promptVisibleAtDepth(course, prompt, depth) {
			projection.dueReviews++
		}
	}

	projection.recommendedLessonID = recommendTeachingLesson(
		course, depth, journey, resumable, projection.progressByLesson,
	)
	if projection.recommendedLessonID != "" {
		projection.recommendedLessonTitle = course.Lessons[projection.recommendedLessonID].Title
		if nodeIndex, exists := nodeIndexes[projection.recommendedLessonID]; exists {
			projection.tree.Nodes[nodeIndex].Recommended = true
		}
	}
	return projection, nil
}

func teachingTreeLessonOrder(course CompiledCourse) []string {
	ordered := make([]string, 0, len(course.Lessons))
	visited := make(map[string]bool, len(course.Lessons))
	var visit func(string)
	visit = func(lessonID string) {
		if visited[lessonID] {
			return
		}
		if _, exists := course.Lessons[lessonID]; !exists {
			return
		}
		visited[lessonID] = true
		ordered = append(ordered, lessonID)
		for _, edge := range course.LessonChildren[lessonID] {
			visit(edge.ToLessonID)
		}
	}
	visit(course.RootLessonID)
	for _, lesson := range course.Pack.Lessons {
		visit(lesson.LessonID)
	}
	return ordered
}

func lessonIDsByPrompt(course CompiledCourse) map[string][]string {
	result := make(map[string][]string, len(course.Prompts))
	for _, lesson := range course.Pack.Lessons {
		for _, activity := range lesson.Activities {
			if activity.PromptID != "" {
				result[activity.PromptID] = append(result[activity.PromptID], lesson.LessonID)
			}
		}
	}
	return result
}

func openingPathItems(course CompiledCourse, lessonIDs []string) []OpeningPathItem {
	items := make([]OpeningPathItem, 0, len(lessonIDs))
	for _, lessonID := range lessonIDs {
		lesson, exists := course.Lessons[lessonID]
		if !exists {
			continue
		}
		items = append(items, OpeningPathItem{LessonID: lessonID, Title: lesson.Title})
	}
	return items
}

func teachingPathLessonIDs(course CompiledCourse, lessonID string) []string {
	if _, exists := course.Lessons[lessonID]; !exists {
		return []string{}
	}
	reversed := []string{lessonID}
	for lessonID != course.RootLessonID {
		edge, exists := course.LessonParent[lessonID]
		if !exists {
			break
		}
		lessonID = edge.FromLessonID
		reversed = append(reversed, lessonID)
	}
	path := make([]string, len(reversed))
	for index := range reversed {
		path[len(reversed)-1-index] = reversed[index]
	}
	return path
}

func validJourneyLessonID(course CompiledCourse, lessonID string) string {
	if _, exists := course.Lessons[lessonID]; exists {
		return lessonID
	}
	return ""
}

func hasVisibleResumable(course CompiledCourse, depth Depth, resumable *StoredSession) bool {
	if resumable == nil || resumable.CourseID != course.Pack.CourseID {
		return false
	}
	if resumable.Mode == OpeningModeReview {
		return true
	}
	lesson, exists := course.Lessons[resumable.LessonID]
	return exists && visibleAtDepth(lesson.MinimumDepth, depth)
}

func recommendTeachingLesson(
	course CompiledCourse,
	depth Depth,
	journey CourseJourney,
	resumable *StoredSession,
	progress map[string]LessonProgress,
) string {
	eligible := func(lessonID string) bool {
		lesson, exists := course.Lessons[lessonID]
		return exists && visibleAtDepth(lesson.MinimumDepth, depth) && !progress[lessonID].Completed
	}
	if resumable != nil && resumable.CourseID == course.Pack.CourseID &&
		resumable.Mode == OpeningModeLesson && eligible(resumable.LessonID) {
		return resumable.LessonID
	}
	if eligible(journey.CurrentLessonID) {
		return journey.CurrentLessonID
	}
	for index := len(journey.PathLessonIDs) - 1; index >= 0; index-- {
		lessonID := journey.PathLessonIDs[index]
		if !progress[lessonID].Completed {
			continue
		}
		for _, edge := range course.LessonChildren[lessonID] {
			if edge.Kind == EdgeContinuation && eligible(edge.ToLessonID) {
				return edge.ToLessonID
			}
		}
	}
	for _, lessonID := range recommendationLessonOrder(course) {
		if eligible(lessonID) {
			return lessonID
		}
	}
	return ""
}

func recommendationLessonOrder(course CompiledCourse) []string {
	ordered := make([]string, 0, len(course.Lessons))
	visited := make(map[string]bool, len(course.Lessons))
	var visit func(string)
	visit = func(lessonID string) {
		if visited[lessonID] {
			return
		}
		if _, exists := course.Lessons[lessonID]; !exists {
			return
		}
		visited[lessonID] = true
		ordered = append(ordered, lessonID)
		children := course.LessonChildren[lessonID]
		for _, edge := range children {
			if edge.Kind == EdgeContinuation {
				visit(edge.ToLessonID)
			}
		}
		for _, edge := range children {
			if edge.Kind != EdgeContinuation {
				visit(edge.ToLessonID)
			}
		}
	}
	visit(course.RootLessonID)
	for _, lesson := range course.Pack.Lessons {
		visit(lesson.LessonID)
	}
	return ordered
}
