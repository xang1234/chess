package openings

import (
	"fmt"
	"sort"
)

func (c *courseCompiler) validateTeachingTree() {
	edgePaths := map[string]string{}
	siblingOrdinals := map[string]map[int]string{}

	for edgeIndex, edge := range c.pack.LessonEdges {
		path := fmt.Sprintf("lessonEdges[%d]", edgeIndex)
		c.registerID(edgePaths, edge.EdgeID, path+".edgeId")
		_, fromOK := c.compiled.Lessons[edge.FromLessonID]
		_, toOK := c.compiled.Lessons[edge.ToLessonID]
		if !fromOK {
			c.addDiagnostic("missing_reference", path+".fromLessonId", fmt.Sprintf("lesson %q does not exist", edge.FromLessonID))
		}
		if !toOK {
			c.addDiagnostic("missing_reference", path+".toLessonId", fmt.Sprintf("lesson %q does not exist", edge.ToLessonID))
		}
		if edge.FromLessonID == edge.ToLessonID && fromOK {
			c.addDiagnostic("lesson_tree_cycle", path+".toLessonId", "lesson edge cannot point to itself")
		}
		if edge.Ordinal <= 0 {
			c.addDiagnostic("invalid_ordinal", path+".ordinal", "lesson edge ordinal must be positive")
		} else {
			byOrdinal := siblingOrdinals[edge.FromLessonID]
			if byOrdinal == nil {
				byOrdinal = map[int]string{}
				siblingOrdinals[edge.FromLessonID] = byOrdinal
			}
			if first, duplicate := byOrdinal[edge.Ordinal]; duplicate {
				c.addDiagnostic("duplicate_ordinal", path+".ordinal", fmt.Sprintf("lesson edge ordinal duplicates %s", first))
			} else {
				byOrdinal[edge.Ordinal] = path
			}
		}
		switch edge.Kind {
		case EdgeContinuation:
		case EdgeAlternative, EdgeReference:
			c.validateText(path+".label", edge.Label, true)
		default:
			c.addDiagnostic("invalid_lesson_edge_kind", path+".kind", fmt.Sprintf("unsupported lesson edge kind %q", edge.Kind))
		}
		edgeRank, edgeDepthOK := depthRank(edge.MinimumDepth)
		if !edgeDepthOK {
			c.addDiagnostic("invalid_depth", path+".minimumDepth", fmt.Sprintf("unsupported depth %q", edge.MinimumDepth))
		}
		if fromOK && toOK {
			fromRank, fromDepthOK := depthRank(c.compiled.Lessons[edge.FromLessonID].MinimumDepth)
			toRank, toDepthOK := depthRank(c.compiled.Lessons[edge.ToLessonID].MinimumDepth)
			if fromDepthOK && toDepthOK && toRank < fromRank {
				c.addDiagnostic("lesson_depth_inversion", path+".toLessonId", "child lesson cannot be shallower than its parent")
			}
			if edgeDepthOK && fromDepthOK && toDepthOK && edgeRank < max(fromRank, toRank) {
				c.addDiagnostic("lesson_edge_depth", path+".minimumDepth", "edge cannot be visible before both endpoint lessons")
			}
		}
	}

	index := buildTeachingTreeIndex(c.compiled)
	for _, lesson := range c.pack.Lessons {
		edges := index.incoming[lesson.LessonID]
		if len(edges) > 1 {
			path := c.basePath(c.lessonPaths, lesson.LessonID, "lesson")
			c.addDiagnostic("lesson_multiple_parents", path+".lessonId", fmt.Sprintf("lesson has %d incoming teaching edges", len(edges)))
		}
	}
	if len(index.roots) != 1 {
		c.addDiagnostic("lesson_tree_root", "lessonEdges", fmt.Sprintf("teaching tree requires one root, found %d", len(index.roots)))
	} else {
		if root, ok := c.compiled.Lessons[index.roots[0]]; ok && root.MinimumDepth != DepthQuick {
			c.addDiagnostic("lesson_tree_root_depth", c.basePath(c.lessonPaths, root.LessonID, "lesson")+".minimumDepth", "teaching tree root must be visible at quick depth")
		}
	}

	c.detectTeachingTreeCycles(index.children)
	if len(index.roots) == 1 {
		c.validateTeachingTreeReachability(index.roots[0], index.children)
	}
	applyTeachingTreeIndex(&c.compiled, index)
}

type teachingTreeIndex struct {
	roots    []string
	incoming map[string][]LessonEdge
	children map[string][]LessonEdge
	parents  map[string]LessonEdge
}

func buildTeachingTreeIndex(compiled CompiledCourse) teachingTreeIndex {
	index := teachingTreeIndex{
		roots: []string{}, incoming: map[string][]LessonEdge{},
		children: map[string][]LessonEdge{}, parents: map[string]LessonEdge{},
	}
	for _, edge := range compiled.Pack.LessonEdges {
		if _, fromOK := compiled.Lessons[edge.FromLessonID]; !fromOK {
			continue
		}
		if _, toOK := compiled.Lessons[edge.ToLessonID]; !toOK {
			continue
		}
		index.children[edge.FromLessonID] = append(index.children[edge.FromLessonID], edge)
		index.incoming[edge.ToLessonID] = append(index.incoming[edge.ToLessonID], edge)
	}
	for lessonID := range index.children {
		sort.SliceStable(index.children[lessonID], func(left, right int) bool {
			children := index.children[lessonID]
			if children[left].Ordinal != children[right].Ordinal {
				return children[left].Ordinal < children[right].Ordinal
			}
			return children[left].EdgeID < children[right].EdgeID
		})
	}
	for lessonID, incoming := range index.incoming {
		if len(incoming) == 1 {
			index.parents[lessonID] = incoming[0]
		}
	}
	for _, lesson := range compiled.Pack.Lessons {
		if len(index.incoming[lesson.LessonID]) == 0 {
			index.roots = append(index.roots, lesson.LessonID)
		}
	}
	return index
}

func applyTeachingTreeIndex(compiled *CompiledCourse, index teachingTreeIndex) {
	compiled.RootLessonID = ""
	if len(index.roots) == 1 {
		compiled.RootLessonID = index.roots[0]
	}
	compiled.LessonChildren = index.children
	compiled.LessonParent = index.parents
}

func indexCompiledTeachingTree(compiled *CompiledCourse) {
	applyTeachingTreeIndex(compiled, buildTeachingTreeIndex(*compiled))
}

func (c *courseCompiler) detectTeachingTreeCycles(children map[string][]LessonEdge) {
	colors := map[string]uint8{}
	var visit func(string)
	visit = func(lessonID string) {
		colors[lessonID] = 1
		for _, edge := range children[lessonID] {
			switch colors[edge.ToLessonID] {
			case 0:
				visit(edge.ToLessonID)
			case 1:
				c.addDiagnostic("lesson_tree_cycle", c.edgePath(edge.EdgeID)+".toLessonId", fmt.Sprintf("edge %q closes a teaching-tree cycle", edge.EdgeID))
			}
		}
		colors[lessonID] = 2
	}
	for _, lesson := range c.pack.Lessons {
		if colors[lesson.LessonID] == 0 {
			visit(lesson.LessonID)
		}
	}
}

func (c *courseCompiler) validateTeachingTreeReachability(rootID string, children map[string][]LessonEdge) {
	for _, depth := range []Depth{DepthQuick, DepthStandard, DepthReference} {
		selectedRank, _ := depthRank(depth)
		visited := map[string]bool{rootID: true}
		queue := []string{rootID}
		for len(queue) > 0 {
			lessonID := queue[0]
			queue = queue[1:]
			for _, edge := range children[lessonID] {
				edgeRank, edgeDepthOK := depthRank(edge.MinimumDepth)
				lesson, lessonOK := c.compiled.Lessons[edge.ToLessonID]
				lessonRank, lessonDepthOK := depthRank(lesson.MinimumDepth)
				if !edgeDepthOK || !lessonOK || !lessonDepthOK || edgeRank > selectedRank || lessonRank > selectedRank {
					continue
				}
				if !visited[edge.ToLessonID] {
					visited[edge.ToLessonID] = true
					queue = append(queue, edge.ToLessonID)
				}
			}
		}
		for _, lesson := range c.pack.Lessons {
			lessonRank, ok := depthRank(lesson.MinimumDepth)
			if ok && lessonRank <= selectedRank && !visited[lesson.LessonID] {
				path := c.basePath(c.lessonPaths, lesson.LessonID, "lesson")
				c.addDiagnostic("lesson_depth_route", path+".minimumDepth", fmt.Sprintf("lesson is unreachable in the %s teaching route", depth))
			}
		}
	}
}

func (c *courseCompiler) edgePath(edgeID string) string {
	for index, edge := range c.pack.LessonEdges {
		if edge.EdgeID == edgeID {
			return fmt.Sprintf("lessonEdges[%d]", index)
		}
	}
	return "lessonEdge"
}
