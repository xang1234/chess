package domain

type Color string

const (
	White Color = "white"
	Black Color = "black"
)

type MoveNode struct {
	UCI      string     `json:"uci"`
	Children []MoveNode `json:"children,omitempty"`
}

type AppliedMove struct {
	UCI          string `json:"uci"`
	ResultingFEN string `json:"resultingFen"`
}
