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

type OpeningCourseSummary struct {
	CourseID         string                  `json:"courseId"`
	Title            string                  `json:"title"`
	Perspective      Perspective             `json:"perspective"`
	Depth            Depth                   `json:"depth"`
	RootPositionID   string                  `json:"rootPositionId"`
	CompletedLessons int                     `json:"completedLessons"`
	TotalLessons     int                     `json:"totalLessons"`
	DueReviews       int                     `json:"dueReviews"`
	NextLessonID     string                  `json:"nextLessonId,omitempty"`
	NextLessonTitle  string                  `json:"nextLessonTitle,omitempty"`
	HasResumable     bool                    `json:"hasResumable"`
	Chapters         []OpeningChapterSummary `json:"chapters"`
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
	Depth        Depth                `json:"depth"`
	Current      *OpeningStepView     `json:"current,omitempty"`
	Summary      *OpeningSummary      `json:"summary,omitempty"`
	Notice       string               `json:"notice,omitempty"`
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

type OpeningStepResult struct {
	Session       OpeningSessionView   `json:"session"`
	StepCompleted bool                 `json:"stepCompleted"`
	Feedback      MoveFeedback         `json:"feedback,omitempty"`
	Message       string               `json:"message,omitempty"`
	AppliedMoves  []domain.AppliedMove `json:"appliedMoves,omitempty"`
	FinalFEN      string               `json:"finalFen,omitempty"`
}

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
