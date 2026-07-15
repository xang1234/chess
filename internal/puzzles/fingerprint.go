package puzzles

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"chess-trainer/internal/domain"
)

func normalizeNodes(nodes []domain.MoveNode) []domain.MoveNode {
	out := make([]domain.MoveNode, len(nodes))
	for i, node := range nodes {
		out[i] = domain.MoveNode{
			UCI:      strings.ToLower(strings.TrimSpace(node.UCI)),
			Children: normalizeNodes(node.Children),
		}
	}
	return out
}

func CoreFingerprint(p PuzzleCore) (string, error) {
	payload := struct {
		FEN      string            `json:"fen"`
		Solver   domain.Color      `json:"solver"`
		Solution []domain.MoveNode `json:"solution"`
	}{strings.TrimSpace(p.DisplayedFEN), p.Solver, normalizeNodes(p.Solution)}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}
