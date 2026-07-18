package puzzles

import (
	"errors"
	"fmt"
	"strings"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/domain"

	"github.com/corentings/chess/v2"
)

const (
	maxSolutionDepth = 256
	maxSolutionNodes = 512
)

func normalizeFEN(_ chessrules.Rules, fen string) (string, error) {
	fields := strings.Fields(fen)
	if len(fields) != 6 {
		return "", errors.New("displayed FEN must contain six fields")
	}
	castlingRights, err := normalizeCastlingRights(fields[2])
	if err != nil {
		return "", err
	}
	fields[2] = castlingRights
	option, err := chess.FEN(strings.Join(fields, " "))
	if err != nil {
		return "", fmt.Errorf("normalize displayed FEN: %w", err)
	}
	return chess.NewGame(option).Position().String(), nil
}

func normalizeCastlingRights(rights string) (string, error) {
	if rights == "-" {
		return rights, nil
	}
	seen := make(map[rune]struct{})
	for _, right := range rights {
		switch right {
		case 'K', 'Q', 'k', 'q':
		default:
			return "", fmt.Errorf("invalid castling rights %q", rights)
		}
		if _, duplicate := seen[right]; duplicate {
			return "", fmt.Errorf("invalid castling rights %q", rights)
		}
		seen[right] = struct{}{}
	}
	if len(seen) == 0 {
		return "", fmt.Errorf("invalid castling rights %q", rights)
	}
	var normalized strings.Builder
	for _, right := range "KQkq" {
		if _, present := seen[right]; present {
			normalized.WriteRune(right)
		}
	}
	return normalized.String(), nil
}

func solverFromFEN(fen string) (domain.Color, error) {
	fields := strings.Fields(fen)
	if len(fields) < 2 {
		return "", errors.New("displayed FEN has no active-color field")
	}
	switch fields[1] {
	case "w":
		return domain.White, nil
	case "b":
		return domain.Black, nil
	default:
		return "", fmt.Errorf("invalid active color %q", fields[1])
	}
}

func linearSolution(moves []string) ([]domain.MoveNode, error) {
	if len(moves) > maxSolutionDepth {
		return nil, fmt.Errorf(
			"solution depth %d exceeds maximum of %d",
			len(moves),
			maxSolutionDepth,
		)
	}
	if len(moves) > maxSolutionNodes {
		return nil, fmt.Errorf("solution exceeds maximum of %d nodes", maxSolutionNodes)
	}

	var solution []domain.MoveNode
	for index := len(moves) - 1; index >= 0; index-- {
		solution = []domain.MoveNode{{
			UCI:      strings.ToLower(moves[index]),
			Children: solution,
		}}
	}
	return solution, nil
}

func normalizeSolutionTree(
	rules chessrules.Rules,
	displayedFEN string,
	nodes []domain.MoveNode,
) ([]domain.MoveNode, int, error) {
	if len(nodes) == 0 {
		return nil, 0, errors.New("solution is empty")
	}
	total := 0
	normalized, depth, err := normalizeSolutionLevel(rules, displayedFEN, nodes, 1, &total)
	if err != nil {
		return nil, 0, err
	}
	return normalized, depth, nil
}

func normalizeSolutionLevel(
	rules chessrules.Rules,
	parentFEN string,
	nodes []domain.MoveNode,
	depth int,
	total *int,
) ([]domain.MoveNode, int, error) {
	seen := make(map[string]struct{})
	normalized := make([]domain.MoveNode, 0)
	maximumDepth := depth
	for index, node := range nodes {
		uci := strings.ToLower(strings.TrimSpace(node.UCI))
		if uci == "" {
			return nil, 0, fmt.Errorf("solution depth %d sibling %d has empty UCI", depth, index)
		}
		if _, duplicate := seen[uci]; duplicate {
			return nil, 0, fmt.Errorf("solution depth %d has duplicate sibling UCI %q", depth, uci)
		}
		seen[uci] = struct{}{}

		*total++
		if *total > maxSolutionNodes {
			return nil, 0, fmt.Errorf("solution exceeds maximum of %d nodes", maxSolutionNodes)
		}
		if depth > maxSolutionDepth {
			return nil, 0, fmt.Errorf("solution depth %d exceeds maximum of %d", depth, maxSolutionDepth)
		}

		nextFEN, err := rules.ApplyUCI(parentFEN, uci)
		if err != nil {
			return nil, 0, fmt.Errorf("solution depth %d sibling %d move %q: %w", depth, index, uci, err)
		}
		normalizedNode := domain.MoveNode{UCI: uci}
		if len(node.Children) > 0 {
			children, childDepth, err := normalizeSolutionLevel(
				rules,
				nextFEN,
				node.Children,
				depth+1,
				total,
			)
			if err != nil {
				return nil, 0, err
			}
			normalizedNode.Children = children
			if childDepth > maximumDepth {
				maximumDepth = childDepth
			}
		}
		normalized = append(normalized, normalizedNode)
	}
	return normalized, maximumDepth, nil
}

func finalizeCore(
	rules chessrules.Rules,
	displayedFEN string,
	solver domain.Color,
	nodes []domain.MoveNode,
) (PuzzleCore, error) {
	normalizedFEN, err := normalizeFEN(rules, displayedFEN)
	if err != nil {
		return PuzzleCore{}, err
	}
	activeColor, err := solverFromFEN(normalizedFEN)
	if err != nil {
		return PuzzleCore{}, err
	}
	if solver != activeColor {
		return PuzzleCore{}, fmt.Errorf(
			"solver %q does not match displayed FEN active color %q",
			solver,
			activeColor,
		)
	}
	normalizedSolution, solutionPlies, err := normalizeSolutionTree(rules, normalizedFEN, nodes)
	if err != nil {
		return PuzzleCore{}, err
	}
	core := PuzzleCore{
		DisplayedFEN:  normalizedFEN,
		Solver:        solver,
		Solution:      normalizedSolution,
		SolutionPlies: solutionPlies,
	}
	core.Fingerprint, err = CoreFingerprint(core)
	if err != nil {
		return PuzzleCore{}, fmt.Errorf("fingerprint puzzle core: %w", err)
	}
	return core, nil
}
