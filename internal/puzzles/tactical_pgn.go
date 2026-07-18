package puzzles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/domain"

	"github.com/corentings/chess/v2"
)

const (
	maxTacticalPGNGameBytes     = 64 * 1024
	maxTacticalPGNMetadataBytes = 64 * 1024
	maxTacticalPGNTags          = 128
)

var tacticalPGNProvenanceTags = []string{
	"Event",
	"Site",
	"Date",
	"Round",
	"Annotator",
}

type tacticalPGNAdapter struct {
	rules chessrules.Rules
}

func NewTacticalPGNAdapter(rules chessrules.Rules) PuzzleAdapter {
	return tacticalPGNAdapter{rules: rules}
}

func (tacticalPGNAdapter) Format() ImportFormat {
	return FormatTacticalPGN
}

func (a tacticalPGNAdapter) Inspect(
	ctx context.Context,
	path string,
) (ImportInspection, bool, error) {
	if err := ctx.Err(); err != nil {
		return ImportInspection{}, false, err
	}
	file, err := os.Open(path)
	if err != nil {
		return ImportInspection{}, false, err
	}
	defer file.Close()

	scanner := chess.NewScanner(contextReader{ctx: ctx, reader: file})
	for {
		scanned, err := scanner.ScanGame()
		if errors.Is(err, io.EOF) {
			return ImportInspection{}, false, nil
		}
		if err != nil {
			return ImportInspection{}, false, fmt.Errorf("inspect tactical PGN: scan game: %w", err)
		}
		if len(scanned.Raw) > maxTacticalPGNGameBytes {
			continue
		}
		_, tags, _ := tokenizeTacticalPGNGame(scanned)
		if !tacticalPGNHasNonEmptyTag(tags, "FEN") {
			continue
		}

		sourceID := firstTacticalPGNTag(tags, "SourceId")
		origin := SourceIDEmbedded
		if sourceID == "" {
			sourceID = path
			origin = SourceIDPath
		}
		return ImportInspection{
			SourceID:       sourceID,
			SourceIDOrigin: origin,
		}, true, nil
	}
}

func tacticalPGNHasNonEmptyTag(tags map[string][]string, name string) bool {
	for _, value := range tags[name] {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func firstTacticalPGNTag(tags map[string][]string, name string) string {
	values := tags[name]
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func (a tacticalPGNAdapter) NewDecoder(
	reader io.Reader,
	inspection ImportInspection,
) (PuzzleDecoder, error) {
	sourceID := strings.TrimSpace(inspection.SourceID)
	if sourceID == "" {
		return nil, errors.New("tactical PGN source ID is required")
	}
	return &tacticalPGNDecoder{
		rules:          a.rules,
		sourceID:       sourceID,
		sourceIDOrigin: inspection.SourceIDOrigin,
		scanner:        chess.NewScanner(reader),
	}, nil
}

type tacticalPGNDecoder struct {
	rules          chessrules.Rules
	sourceID       string
	sourceIDOrigin SourceIDOrigin
	scanner        *chess.Scanner
	ordinal        int64
	closed         bool
}

func (d *tacticalPGNDecoder) Next(ctx context.Context) (DecodedRecord, error) {
	if err := ctx.Err(); err != nil {
		return DecodedRecord{}, err
	}
	if d.closed {
		return DecodedRecord{}, io.EOF
	}

	scanned, err := d.scanner.ScanGame()
	if errors.Is(err, io.EOF) {
		return DecodedRecord{}, io.EOF
	}
	nextOrdinal := d.ordinal + 1
	if err != nil {
		return DecodedRecord{}, fmt.Errorf("scan PGN game %d: %w", nextOrdinal, err)
	}
	d.ordinal = nextOrdinal
	if len(scanned.Raw) > maxTacticalPGNGameBytes {
		return tacticalPGNRejection(
			d.ordinal,
			fmt.Errorf("PGN game exceeds maximum of %d bytes", maxTacticalPGNGameBytes),
		), nil
	}

	tokens, tags, tokenizeErr := tokenizeTacticalPGNGame(scanned)
	recordSourceIDs, hasSourceID := tags["SourceId"]
	if d.ordinal == 1 && d.sourceIDOrigin == SourceIDEmbedded && !hasSourceID {
		return DecodedRecord{}, fmt.Errorf(
			"PGN game 1 does not reproduce inspected SourceId %q",
			d.sourceID,
		)
	}
	for _, value := range recordSourceIDs {
		recordSourceID := strings.TrimSpace(value)
		firstGameFallback := d.ordinal == 1 &&
			d.sourceIDOrigin == SourceIDPath &&
			recordSourceID == ""
		if recordSourceID != d.sourceID && !firstGameFallback {
			return DecodedRecord{}, fmt.Errorf(
				"PGN game %d SourceId %q conflicts with inspected source ID %q",
				d.ordinal,
				recordSourceID,
				d.sourceID,
			)
		}
	}
	if tokenizeErr != nil {
		return tacticalPGNRejection(d.ordinal, tokenizeErr), nil
	}
	game, err := parseTacticalPGNTokens(tokens)
	if err != nil {
		return tacticalPGNRejection(d.ordinal, err), nil
	}

	puzzle, err := d.normalizeGame(d.ordinal, game, tags)
	if err != nil {
		return tacticalPGNRejection(d.ordinal, err), nil
	}
	return DecodedRecord{Puzzle: &puzzle}, nil
}

func (d *tacticalPGNDecoder) Close() error {
	d.closed = true
	return nil
}

func tokenizeTacticalPGNGame(
	scanned *chess.GameScanned,
) ([]chess.Token, map[string][]string, error) {
	tokens, err := chess.TokenizeGame(scanned)
	if err != nil {
		return nil, nil, fmt.Errorf("tokenize PGN game: %w", err)
	}
	tags := make(map[string][]string)
	tagCount := 0
	for position := 0; position+3 < len(tokens); position += 4 {
		if tokens[position].Type != chess.TagStart ||
			tokens[position+1].Type != chess.TagKey ||
			tokens[position+2].Type != chess.TagValue ||
			tokens[position+3].Type != chess.TagEnd {
			break
		}
		key := tokens[position+1].Value
		tags[key] = append(tags[key], tokens[position+2].Value)
		tagCount++
	}
	if tagCount > maxTacticalPGNTags {
		return tokens, tags, fmt.Errorf(
			"PGN game has %d tag pairs, maximum is %d",
			tagCount,
			maxTacticalPGNTags,
		)
	}
	for _, token := range tokens {
		if token.Error != nil {
			return tokens, tags, fmt.Errorf("tokenize PGN game: %w", token.Error)
		}
	}
	return tokens, tags, nil
}

func parseTacticalPGNTokens(tokens []chess.Token) (*chess.Game, error) {
	game, err := chess.NewParser(tokens).Parse()
	if err != nil {
		return nil, fmt.Errorf("parse PGN game: %w", err)
	}
	return game, nil
}

func (d *tacticalPGNDecoder) normalizeGame(
	ordinal int64,
	game *chess.Game,
	tags map[string][]string,
) (TrainingPuzzle, error) {
	if _, present := tags["FEN"]; !present {
		return TrainingPuzzle{}, errors.New("FEN is required")
	}
	sourceFEN := strings.TrimSpace(game.GetTagPair("FEN"))
	if sourceFEN == "" {
		return TrainingPuzzle{}, errors.New("FEN is required")
	}
	if _, present := tags["SetUp"]; present && strings.TrimSpace(game.GetTagPair("SetUp")) != "1" {
		return TrainingPuzzle{}, errors.New(`SetUp must be "1" when present`)
	}

	var solver domain.Color
	whiteSolver := strings.EqualFold(strings.TrimSpace(game.GetTagPair("White")), "solver")
	blackSolver := strings.EqualFold(strings.TrimSpace(game.GetTagPair("Black")), "solver")
	if whiteSolver == blackSolver {
		return TrainingPuzzle{}, errors.New(`exactly one of White or Black must equal "solver"`)
	}
	if whiteSolver {
		solver = domain.White
	} else {
		solver = domain.Black
	}

	normalizedSourceFEN, err := normalizeFEN(d.rules, sourceFEN)
	if err != nil {
		return TrainingPuzzle{}, fmt.Errorf("normalize source FEN: %w", err)
	}
	history := game.MoveHistory()
	moves := make([]string, len(history))
	notation := chess.UCINotation{}
	for index, item := range history {
		moves[index] = notation.Encode(item.PrePosition, item.Move)
	}

	displayedFEN := normalizedSourceFEN
	preludeUCI := ""
	initialColor, err := solverFromFEN(normalizedSourceFEN)
	if err != nil {
		return TrainingPuzzle{}, err
	}
	if initialColor != solver {
		if len(history) == 0 {
			return TrainingPuzzle{}, errors.New("at least one solution move must remain after the prelude")
		}
		preludeUCI = moves[0]
		displayedFEN = history[0].PostPosition.String()
		moves = moves[1:]
	}
	if len(moves) == 0 {
		return TrainingPuzzle{}, errors.New("at least one solution move must remain after the prelude")
	}

	solution, err := linearSolution(moves)
	if err != nil {
		return TrainingPuzzle{}, fmt.Errorf("validate PGN move line: %w", err)
	}
	core, err := finalizeCore(d.rules, displayedFEN, solver, solution)
	if err != nil {
		return TrainingPuzzle{}, fmt.Errorf("validate PGN move line: %w", err)
	}
	externalID := strings.TrimSpace(game.GetTagPair("PuzzleId"))
	if externalID == "" {
		externalID = strconv.FormatInt(ordinal, 10)
	}
	metadata := tacticalPGNMetadata(game)

	return TrainingPuzzle{
		Core: core,
		Occurrence: PuzzleOccurrence{
			SourceID:   d.sourceID,
			SourceKind: string(FormatTacticalPGN),
			ExternalID: externalID,
			SourceFEN:  normalizedSourceFEN,
			PreludeUCI: preludeUCI,
			Metadata:   metadata,
			Ordinal:    ordinal,
		},
	}, nil
}

func tacticalPGNMetadata(game *chess.Game) map[string]any {
	metadata := make(map[string]any)
	for _, name := range tacticalPGNProvenanceTags {
		value := strings.TrimSpace(game.GetTagPair(name))
		if value == "" {
			continue
		}
		metadata[name] = value
		encoded, err := json.Marshal(metadata)
		if err != nil || len(encoded) > maxTacticalPGNMetadataBytes {
			delete(metadata, name)
		}
	}
	return metadata
}

func tacticalPGNRejection(ordinal int64, err error) DecodedRecord {
	rejection := Rejection{Ordinal: ordinal, Reason: err.Error()}
	return DecodedRecord{Rejection: &rejection}
}
