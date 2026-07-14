package chessrules

import "github.com/corentings/chess/v2"

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
