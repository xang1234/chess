package openings

import "chess-trainer/internal/domain"

type MoveFeedback string

const (
	FeedbackExpected    MoveFeedback = "expected"
	FeedbackAlternative MoveFeedback = "alternative"
	FeedbackOffCourse   MoveFeedback = "off_course"
)

type OpeningHomeView struct {
	Notice  string                 `json:"notice,omitempty"`
	Courses []OpeningCourseSummary `json:"courses"`
}

type OpeningNodeProgress string

const (
	NodeAvailable  OpeningNodeProgress = "available"
	NodeInProgress OpeningNodeProgress = "in_progress"
	NodeCompleted  OpeningNodeProgress = "completed"
)

type OpeningTeachingNodeView struct {
	LessonID            string              `json:"lessonId"`
	ChapterID           string              `json:"chapterId"`
	Title               string              `json:"title"`
	Objective           string              `json:"objective"`
	MinimumDepth        Depth               `json:"minimumDepth"`
	Progress            OpeningNodeProgress `json:"progress"`
	CompletedActivities int                 `json:"completedActivities"`
	RequiredActivities  int                 `json:"requiredActivities"`
	Recommended         bool                `json:"recommended"`
	ReviewDue           bool                `json:"reviewDue"`
	Visible             bool                `json:"visible"`
}

type OpeningTeachingEdgeView struct {
	EdgeID       string         `json:"edgeId"`
	FromLessonID string         `json:"fromLessonId"`
	ToLessonID   string         `json:"toLessonId"`
	Ordinal      int            `json:"ordinal"`
	Kind         LessonEdgeKind `json:"kind"`
	Label        string         `json:"label,omitempty"`
	MinimumDepth Depth          `json:"minimumDepth"`
}

type OpeningTeachingTreeView struct {
	RootLessonID string                    `json:"rootLessonId"`
	Nodes        []OpeningTeachingNodeView `json:"nodes"`
	Edges        []OpeningTeachingEdgeView `json:"edges"`
}

type OpeningPathItem struct {
	LessonID string `json:"lessonId"`
	Title    string `json:"title"`
}

type OpeningCourseSummary struct {
	CourseID               string                  `json:"courseId"`
	Title                  string                  `json:"title"`
	Perspective            Perspective             `json:"perspective"`
	Depth                  Depth                   `json:"depth"`
	RootPositionID         string                  `json:"rootPositionId"`
	CompletedLessons       int                     `json:"completedLessons"`
	TotalLessons           int                     `json:"totalLessons"`
	DueReviews             int                     `json:"dueReviews"`
	NextLessonID           string                  `json:"nextLessonId,omitempty"`
	NextLessonTitle        string                  `json:"nextLessonTitle,omitempty"`
	CurrentLessonID        string                  `json:"currentLessonId,omitempty"`
	CurrentActivityID      string                  `json:"currentActivityId,omitempty"`
	CurrentPath            []OpeningPathItem       `json:"currentPath"`
	RecommendedLessonID    string                  `json:"recommendedLessonId,omitempty"`
	RecommendedLessonTitle string                  `json:"recommendedLessonTitle,omitempty"`
	HasResumable           bool                    `json:"hasResumable"`
	HasResumableReview     bool                    `json:"hasResumableReview"`
	Tree                   OpeningTeachingTreeView `json:"tree"`
	Chapters               []OpeningChapterSummary `json:"chapters"`
}

type OpeningChapterSummary struct {
	ChapterID string                 `json:"chapterId"`
	Title     string                 `json:"title"`
	Lessons   []OpeningLessonSummary `json:"lessons"`
}

type OpeningLessonSummary struct {
	LessonID       string `json:"lessonId"`
	Title          string `json:"title"`
	CompletedSteps int    `json:"completedSteps"`
	TotalSteps     int    `json:"totalSteps"`
	Completed      bool   `json:"completed"`
}

type OpeningSessionView struct {
	SessionID    string               `json:"sessionId"`
	Mode         OpeningSessionMode   `json:"mode"`
	Status       OpeningSessionStatus `json:"status"`
	CourseID     string               `json:"courseId"`
	GenerationID string               `json:"generationId"`
	LessonID     string               `json:"lessonId"`
	CourseTitle  string               `json:"courseTitle"`
	Path         []OpeningPathItem    `json:"path"`
	Depth        Depth                `json:"depth"`
	Current      *OpeningActivityView `json:"current,omitempty"`
	Summary      *OpeningSummary      `json:"summary,omitempty"`
	Notice       string               `json:"notice,omitempty"`
}

type OpeningActivityLine struct {
	Label string   `json:"label"`
	Moves []string `json:"moves"`
}

type OpeningActivityView struct {
	ActivityID         string                    `json:"activityId"`
	Kind               ActivityKind              `json:"kind"`
	Title              string                    `json:"title"`
	Instruction        string                    `json:"instruction"`
	Required           bool                      `json:"required"`
	VariationName      string                    `json:"variationName,omitempty"`
	PositionID         string                    `json:"positionId,omitempty"`
	CurrentFEN         string                    `json:"currentFen"`
	Orientation        Perspective               `json:"orientation"`
	LegalMoves         []string                  `json:"legalMoves"`
	TeachingNoteTexts  []string                  `json:"teachingNoteTexts"`
	ReferenceNoteTexts []string                  `json:"referenceNoteTexts"`
	Comparison         []OpeningActivityLine     `json:"comparison"`
	Annotations        []BoardAnnotation         `json:"annotations"`
	MovesToHere        []domain.AppliedMove      `json:"movesToHere"`
	ActivityNumber     int                       `json:"activityNumber"`
	ActivityTotal      int                       `json:"activityTotal"`
	CompletedIdeas     int                       `json:"completedIdeas"`
	RequiredIdeas      int                       `json:"requiredIdeas"`
	HintLevel          int                       `json:"hintLevel"`
	CanReveal          bool                      `json:"canReveal"`
	ReferenceSections  []OpeningReferenceSection `json:"referenceSections"`
}

type OpeningReferenceSection struct {
	ActivityID  string            `json:"activityId"`
	Title       string            `json:"title"`
	Instruction string            `json:"instruction"`
	PositionID  string            `json:"positionId,omitempty"`
	NoteTexts   []string          `json:"noteTexts"`
	Annotations []BoardAnnotation `json:"annotations"`
}

type OpeningRoadmapCheckpoint struct {
	CompletedLessonID      string            `json:"completedLessonId"`
	Path                   []OpeningPathItem `json:"path"`
	AvailableLessonIDs     []string          `json:"availableLessonIds"`
	RecommendedLessonID    string            `json:"recommendedLessonId,omitempty"`
	RecommendedLessonTitle string            `json:"recommendedLessonTitle,omitempty"`
	CompletedLessons       int               `json:"completedLessons"`
	TotalLessons           int               `json:"totalLessons"`
}

type OpeningStepView struct {
	StepID             string      `json:"stepId"`
	Kind               StepKind    `json:"kind"`
	Title              string      `json:"title"`
	Instruction        string      `json:"instruction"`
	VariationName      string      `json:"variationName,omitempty"`
	PositionID         string      `json:"positionId"`
	CurrentFEN         string      `json:"currentFen"`
	Orientation        Perspective `json:"orientation"`
	LegalMoves         []string    `json:"legalMoves"`
	NoteTexts          []string    `json:"noteTexts"`
	ReferenceNoteTexts []string    `json:"referenceNoteTexts"`
	StepNumber         int         `json:"stepNumber"`
	StepTotal          int         `json:"stepTotal"`
	HintLevel          int         `json:"hintLevel"`
	CanReveal          bool        `json:"canReveal"`
}

type OpeningSummary struct {
	TotalPrompts       int `json:"totalPrompts"`
	PositionsRecalled  int `json:"positionsRecalled"`
	BranchesRecognized int `json:"branchesRecognized"`
	Retried            int `json:"retried"`
	UsedHint           int `json:"usedHint"`
	Revealed           int `json:"revealed"`
}

type OpeningActivityResult struct {
	Session           OpeningSessionView        `json:"session"`
	ActivityCompleted bool                      `json:"activityCompleted"`
	StepCompleted     bool                      `json:"stepCompleted,omitempty"`
	Feedback          MoveFeedback              `json:"feedback,omitempty"`
	Message           string                    `json:"message,omitempty"`
	AppliedMoves      []domain.AppliedMove      `json:"appliedMoves,omitempty"`
	FinalFEN          string                    `json:"finalFen,omitempty"`
	Checkpoint        *OpeningRoadmapCheckpoint `json:"checkpoint,omitempty"`
}

type OpeningStepResult = OpeningActivityResult

type OpeningHintResult struct {
	Session      OpeningSessionView `json:"session"`
	Level        int                `json:"level"`
	Text         string             `json:"text"`
	SourceSquare string             `json:"sourceSquare,omitempty"`
	TargetSquare string             `json:"targetSquare,omitempty"`
	CanReveal    bool               `json:"canReveal"`
}

type ExplorerMove struct {
	MoveID        string       `json:"moveId"`
	UCI           string       `json:"uci"`
	SAN           string       `json:"san"`
	ToPositionID  string       `json:"toPositionId"`
	Role          TrainingRole `json:"role"`
	VariationName string       `json:"variationName,omitempty"`
	Evaluation    Evaluation   `json:"evaluation"`
	SourceRef     SourceRef    `json:"sourceRef"`
}

type NoteView struct {
	Kind      string    `json:"kind"`
	Text      string    `json:"text"`
	SourceRef SourceRef `json:"sourceRef"`
}

type ExplorerPositionView struct {
	CourseID      string         `json:"courseId"`
	PositionID    string         `json:"positionId"`
	FEN           string         `json:"fen"`
	Label         string         `json:"label"`
	Evaluation    Evaluation     `json:"evaluation"`
	Notes         []NoteView     `json:"notes"`
	Moves         []ExplorerMove `json:"moves"`
	IncomingPaths int            `json:"incomingPaths"`
}
