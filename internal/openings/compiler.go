package openings

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type RulesPort interface {
	ApplyUCI(string, string) (string, error)
	SAN(string, string) (string, error)
	LegalMoves(string) ([]string, error)
}

type Diagnostic struct {
	Code    string `json:"code"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

type ValidationError struct {
	Diagnostics []Diagnostic
}

func (e *ValidationError) Error() string {
	if e == nil || len(e.Diagnostics) == 0 {
		return "opening course validation failed"
	}
	lines := make([]string, len(e.Diagnostics))
	for index, diagnostic := range e.Diagnostics {
		lines[index] = fmt.Sprintf("%s %s: %s", diagnostic.Code, diagnostic.Path, diagnostic.Message)
	}
	return strings.Join(lines, "\n")
}

type CompiledPosition struct {
	ID         string
	FEN        string
	Label      string
	Evaluation Evaluation
	NoteIDs    []string
}

type CompiledMove struct {
	Move
	SAN string
}

type CompiledPrompt struct {
	Prompt
	SemanticFingerprint string
}

type CoverageItem struct {
	CoverageID  string   `json:"coverageId"`
	PrintedPage int      `json:"printedPage"`
	TableColumn string   `json:"tableColumn,omitempty"`
	NoteLabel   string   `json:"noteLabel,omitempty"`
	RecordIDs   []string `json:"recordIds"`
}

type CoverageReport struct {
	Expected   []string       `json:"expected"`
	Captured   []CoverageItem `json:"captured"`
	Missing    []string       `json:"missing"`
	Unexpected []string       `json:"unexpected"`
}

type CompiledCourse struct {
	Pack      CoursePack
	Positions map[string]CompiledPosition
	Moves     map[string]CompiledMove
	Notes     map[string]Note
	Chapters  map[string]Chapter
	Lessons   map[string]Lesson
	Prompts   map[string]CompiledPrompt
	Outgoing  map[string][]string
	Incoming  map[string][]string
	Coverage  CoverageReport
}

func (c CompiledCourse) VisibleMoves(positionID string, depth Depth) []CompiledMove {
	selectedRank, ok := depthRank(depth)
	if !ok {
		return []CompiledMove{}
	}
	moveIDs := c.Outgoing[positionID]
	visible := make([]CompiledMove, 0, len(moveIDs))
	for _, moveID := range moveIDs {
		move, exists := c.Moves[moveID]
		if !exists {
			continue
		}
		minimumRank, valid := depthRank(move.MinimumDepth)
		if valid && minimumRank <= selectedRank {
			visible = append(visible, move)
		}
	}
	return visible
}

type courseCompiler struct {
	pack             CoursePack
	rules            RulesPort
	compiled         CompiledCourse
	diagnostics      []Diagnostic
	positionPaths    map[string]string
	movePaths        map[string]string
	notePaths        map[string]string
	chapterPaths     map[string]string
	lessonPaths      map[string]string
	promptPaths      map[string]string
	reachableByDepth map[Depth]map[string]bool
}

var stableIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)

func Compile(pack CoursePack, rules RulesPort) (CompiledCourse, error) {
	normalized, err := NormalizeCoursePack(pack)
	if err != nil {
		return CompiledCourse{}, err
	}
	compiler := newCourseCompiler(normalized, rules)
	compiler.indexAndValidateValues()
	compiler.validateReferences()
	compiler.compileGraph()
	compiler.validateDepthsAndRoles()
	compiler.validateLessons()
	compiler.compileFingerprints()
	compiler.compiled.Coverage, compiler.diagnostics = compileCoverage(normalized, compiler.diagnostics)
	compiler.sortDiagnostics()
	if len(compiler.diagnostics) != 0 {
		return compiler.compiled, &ValidationError{Diagnostics: compiler.diagnostics}
	}
	return compiler.compiled, nil
}

func newCourseCompiler(pack CoursePack, rules RulesPort) *courseCompiler {
	return &courseCompiler{
		pack:  pack,
		rules: rules,
		compiled: CompiledCourse{
			Pack: pack, Positions: map[string]CompiledPosition{}, Moves: map[string]CompiledMove{},
			Notes: map[string]Note{}, Chapters: map[string]Chapter{}, Lessons: map[string]Lesson{},
			Prompts: map[string]CompiledPrompt{}, Outgoing: map[string][]string{}, Incoming: map[string][]string{},
		},
		positionPaths: map[string]string{}, movePaths: map[string]string{}, notePaths: map[string]string{},
		chapterPaths: map[string]string{}, lessonPaths: map[string]string{}, promptPaths: map[string]string{},
		reachableByDepth: map[Depth]map[string]bool{},
	}
}

func (c *courseCompiler) addDiagnostic(code, path, message string) {
	c.diagnostics = append(c.diagnostics, Diagnostic{Code: code, Path: path, Message: message})
}

func (c *courseCompiler) sortDiagnostics() {
	sort.SliceStable(c.diagnostics, func(left, right int) bool {
		a, b := c.diagnostics[left], c.diagnostics[right]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Message < b.Message
	})
}

func depthRank(depth Depth) (int, bool) {
	switch depth {
	case DepthQuick:
		return 0, true
	case DepthStandard:
		return 1, true
	case DepthReference:
		return 2, true
	default:
		return 0, false
	}
}

func CanonicalPosition(fen string) (string, error) {
	fields := strings.Fields(fen)
	if len(fields) < 4 {
		return "", fmt.Errorf("FEN has %d fields, want at least 4", len(fields))
	}
	return strings.Join(fields[:4], " "), nil
}
