package domain

type SessionView struct {
	SessionID    string          `json:"sessionId"`
	Mode         string          `json:"mode"`
	Status       string          `json:"status"`
	CurrentIndex int             `json:"currentIndex"`
	Total        int             `json:"total"`
	Current      *PuzzleView     `json:"current,omitempty"`
	Summary      *SessionSummary `json:"summary,omitempty"`
}

type PuzzleView struct {
	Fingerprint    string `json:"fingerprint"`
	SourceFEN      string `json:"sourceFen,omitempty"`
	DisplayedFEN   string `json:"displayedFen"`
	CurrentFEN     string `json:"currentFen"`
	PreludeUCI     string `json:"preludeUci,omitempty"`
	Solver         Color  `json:"solver"`
	CurrentPath    []int  `json:"currentPath"`
	PuzzleNumber   int    `json:"puzzleNumber"`
	PuzzleTotal    int    `json:"puzzleTotal"`
	HintLevel      int    `json:"hintLevel"`
	IncorrectMoves int    `json:"incorrectMoves"`
	CanReveal      bool   `json:"canReveal"`
}

type MoveResult struct {
	Session         SessionView `json:"session"`
	Correct         bool        `json:"correct"`
	PuzzleCompleted bool        `json:"puzzleCompleted"`
	Message         string      `json:"message,omitempty"`
}

type HintResult struct {
	Level        int    `json:"level"`
	Text         string `json:"text"`
	SourceSquare string `json:"sourceSquare,omitempty"`
	TargetSquare string `json:"targetSquare,omitempty"`
	CanReveal    bool   `json:"canReveal"`
}

type SessionSummary struct {
	Total       int `json:"total"`
	FirstTry    int `json:"firstTry"`
	Retried     int `json:"retried"`
	UsedHint    int `json:"usedHint"`
	Revealed    int `json:"revealed"`
	Unavailable int `json:"unavailable"`
}
