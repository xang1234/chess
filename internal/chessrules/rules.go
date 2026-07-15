package chessrules

import (
	"errors"
	"fmt"

	"github.com/corentings/chess/v2"
)

type Rules struct{}

func gameAt(fen string) (*chess.Game, error) {
	option, err := chess.FEN(fen)
	if err != nil {
		return nil, err
	}
	return chess.NewGame(option), nil
}

func (Rules) ApplyUCI(fen, uci string) (string, error) {
	game, err := gameAt(fen)
	if err != nil {
		return "", err
	}
	if err := game.PushNotationMove(uci, chess.UCINotation{}, nil); err != nil {
		return "", err
	}
	return game.Position().String(), nil
}

// ApplyUCILine validates a complete UCI move line using one game instance and
// returns the position after its first move. Puzzle imports use that first
// position as the solver-facing board while still rejecting an illegal move
// anywhere later in the supplied solution.
func (Rules) ApplyUCILine(fen string, moves []string) (string, error) {
	if len(moves) == 0 {
		return "", errors.New("UCI line is empty")
	}
	game, err := gameAt(fen)
	if err != nil {
		return "", err
	}
	var afterFirst string
	for index, move := range moves {
		if err := game.PushNotationMove(move, chess.UCINotation{}, nil); err != nil {
			return "", fmt.Errorf("move %d %q: %w", index, move, err)
		}
		if index == 0 {
			afterFirst = game.Position().String()
		}
	}
	return afterFirst, nil
}

func (Rules) SAN(fen, uci string) (string, error) {
	game, err := gameAt(fen)
	if err != nil {
		return "", err
	}
	move, err := (chess.UCINotation{}).Decode(game.Position(), uci)
	if err != nil {
		return "", err
	}
	return (chess.AlgebraicNotation{}).Encode(game.Position(), move), nil
}

func (Rules) IsCheckmateMove(fen, uci string) bool {
	game, err := gameAt(fen)
	if err != nil {
		return false
	}
	if err := game.PushNotationMove(uci, chess.UCINotation{}, nil); err != nil {
		return false
	}
	return game.Method() == chess.Checkmate
}
