package openings

type Depth string

const (
	DepthQuick     Depth = "quick"
	DepthStandard  Depth = "standard"
	DepthReference Depth = "reference"
)

type Perspective string

const (
	PerspectiveWhite Perspective = "white"
	PerspectiveBlack Perspective = "black"
)

type TrainingRole string

const (
	RoleRepertoire  TrainingRole = "repertoire"
	RoleOpponent    TrainingRole = "opponent"
	RoleAlternative TrainingRole = "alternative"
)

type StepKind string

const (
	StepExplain StepKind = "explain"
	StepWatch   StepKind = "watch"
	StepTry     StepKind = "try"
	StepBranch  StepKind = "branch"
	StepRecall  StepKind = "recall"
)

type ActivityKind string

const (
	ActivityConcept       ActivityKind = "concept"
	ActivityDemonstration ActivityKind = "demonstration"
	ActivityDecision      ActivityKind = "decision"
	ActivityComparison    ActivityKind = "comparison"
	ActivityRecap         ActivityKind = "recap"
	ActivityReference     ActivityKind = "reference"
)

type LessonEdgeKind string

const (
	EdgeContinuation LessonEdgeKind = "continuation"
	EdgeAlternative  LessonEdgeKind = "alternative"
	EdgeReference    LessonEdgeKind = "reference"
)

type EvaluationCode string

const (
	EvaluationNone         EvaluationCode = "none"
	EvaluationEqual        EvaluationCode = "equal"
	EvaluationUnclear      EvaluationCode = "unclear"
	EvaluationWhiteSlight  EvaluationCode = "white_slight"
	EvaluationBlackSlight  EvaluationCode = "black_slight"
	EvaluationWhiteClear   EvaluationCode = "white_clear"
	EvaluationBlackClear   EvaluationCode = "black_clear"
	EvaluationWhiteWinning EvaluationCode = "white_winning"
	EvaluationBlackWinning EvaluationCode = "black_winning"
)

type Evaluation struct {
	Code         EvaluationCode `json:"code"`
	SourceSymbol string         `json:"sourceSymbol,omitempty"`
}

type SourceRef struct {
	PrintedPage int    `json:"printedPage"`
	TableColumn string `json:"tableColumn,omitempty"`
	NoteLabel   string `json:"noteLabel,omitempty"`
	CoverageID  string `json:"coverageId"`
}

type SourceCoverage struct {
	PrintedPages       []int    `json:"printedPages"`
	ExpectedReferences []string `json:"expectedReferences"`
}

type CourseSource struct {
	Title            string `json:"title"`
	Edition          string `json:"edition"`
	PrivateUseNotice string `json:"privateUseNotice"`
}

type Position struct {
	PositionID string     `json:"positionId"`
	Label      string     `json:"label,omitempty"`
	Evaluation Evaluation `json:"evaluation"`
	NoteIDs    []string   `json:"noteIds"`
}

type Move struct {
	MoveID         string       `json:"moveId"`
	FromPositionID string       `json:"fromPositionId"`
	ToPositionID   string       `json:"toPositionId"`
	UCI            string       `json:"uci"`
	MinimumDepth   Depth        `json:"minimumDepth"`
	TrainingRole   TrainingRole `json:"trainingRole"`
	VariationName  string       `json:"variationName,omitempty"`
	Evaluation     Evaluation   `json:"evaluation"`
	NoteIDs        []string     `json:"noteIds"`
	SourceRef      SourceRef    `json:"sourceRef"`
}

type Note struct {
	NoteID    string    `json:"noteId"`
	Kind      string    `json:"kind"`
	Text      string    `json:"text"`
	SourceRef SourceRef `json:"sourceRef"`
}

type Chapter struct {
	ChapterID    string `json:"chapterId"`
	Ordinal      int    `json:"ordinal"`
	Title        string `json:"title"`
	Overview     string `json:"overview"`
	MinimumDepth Depth  `json:"minimumDepth"`
}

type Lesson struct {
	LessonID        string           `json:"lessonId"`
	ChapterID       string           `json:"chapterId"`
	Ordinal         int              `json:"ordinal"`
	Title           string           `json:"title"`
	Objectives      []string         `json:"objectives"`
	MinimumDepth    Depth            `json:"minimumDepth"`
	StartPositionID string           `json:"startPositionId"`
	Steps           []LessonStep     `json:"steps,omitempty"`
	Activities      []LessonActivity `json:"activities,omitempty"`
}

type LessonStep struct {
	StepID      string   `json:"stepId"`
	Kind        StepKind `json:"kind"`
	PositionID  string   `json:"positionId"`
	Title       string   `json:"title"`
	Instruction string   `json:"instruction"`
	NoteIDs     []string `json:"noteIds"`
	MoveIDs     []string `json:"moveIds"`
	PromptID    string   `json:"promptId,omitempty"`
}

type ActivityLine struct {
	Label   string   `json:"label"`
	MoveIDs []string `json:"moveIds"`
}

type BoardAnnotation struct {
	Kind  string `json:"kind"`
	From  string `json:"from"`
	To    string `json:"to,omitempty"`
	Label string `json:"label,omitempty"`
}

type LessonActivity struct {
	ActivityID  string            `json:"activityId"`
	Kind        ActivityKind      `json:"kind"`
	Title       string            `json:"title"`
	Instruction string            `json:"instruction"`
	Required    bool              `json:"required"`
	PositionID  string            `json:"positionId,omitempty"`
	NoteIDs     []string          `json:"noteIds"`
	MoveIDs     []string          `json:"moveIds"`
	PromptID    string            `json:"promptId,omitempty"`
	Comparison  []ActivityLine    `json:"comparison,omitempty"`
	Annotations []BoardAnnotation `json:"annotations,omitempty"`
}

type LessonEdge struct {
	EdgeID       string         `json:"edgeId"`
	FromLessonID string         `json:"fromLessonId"`
	ToLessonID   string         `json:"toLessonId"`
	Ordinal      int            `json:"ordinal"`
	Kind         LessonEdgeKind `json:"kind"`
	Label        string         `json:"label,omitempty"`
	MinimumDepth Depth          `json:"minimumDepth"`
}

type Prompt struct {
	PromptID                   string   `json:"promptId"`
	PositionID                 string   `json:"positionId"`
	PrimaryMoveID              string   `json:"primaryMoveId"`
	AcceptedAlternativeMoveIDs []string `json:"acceptedAlternativeMoveIds"`
}

type CoursePack struct {
	SchemaVersion  int            `json:"schemaVersion"`
	CourseID       string         `json:"courseId"`
	ContentVersion string         `json:"contentVersion"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	Perspective    Perspective    `json:"perspective"`
	DefaultDepth   Depth          `json:"defaultDepth"`
	RootPositionID string         `json:"rootPositionId"`
	RootFEN        string         `json:"rootFen"`
	Source         CourseSource   `json:"source"`
	SourceCoverage SourceCoverage `json:"sourceCoverage"`
	Positions      []Position     `json:"positions"`
	Moves          []Move         `json:"moves"`
	Notes          []Note         `json:"notes"`
	LessonEdges    []LessonEdge   `json:"lessonEdges,omitempty"`
	Chapters       []Chapter      `json:"chapters"`
	Lessons        []Lesson       `json:"lessons"`
	Prompts        []Prompt       `json:"prompts"`
}
