package domain

type SourceRef struct {
	SourceID    string         `json:"sourceId"`
	ExternalID  string         `json:"externalId,omitempty"`
	URL         string         `json:"url,omitempty"`
	Attribution string         `json:"attribution,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type Puzzle struct {
	Fingerprint  string      `json:"fingerprint"`
	SourceFEN    string      `json:"sourceFen,omitempty"`
	PreludeUCI   string      `json:"preludeUci,omitempty"`
	DisplayedFEN string      `json:"displayedFen"`
	Solver       Color       `json:"solver"`
	Solution     []MoveNode  `json:"solution"`
	Rating       *int        `json:"rating,omitempty"`
	Themes       []string    `json:"themes"`
	Popularity   *int        `json:"popularity,omitempty"`
	PlayCount    *int        `json:"playCount,omitempty"`
	Sources      []SourceRef `json:"sources"`
}
