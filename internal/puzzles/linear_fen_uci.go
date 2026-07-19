package puzzles

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"chess-trainer/internal/chessrules"
)

const maxLinearFENLineBytes = 1 << 20

type linearFENAdapter struct {
	rules chessrules.Rules
}

func NewLinearFENAdapter(rules chessrules.Rules) PuzzleAdapter {
	return linearFENAdapter{rules: rules}
}

func (linearFENAdapter) Descriptor() ImportFormatDescriptor {
	return ImportFormatDescriptor{
		Format: FormatLinearFENUCI, Label: "Linear FEN/UCI",
		CanonicalExtension: ".txt", FileFilterDescription: "FEN/UCI collection",
	}
}

func (a linearFENAdapter) Inspect(
	ctx context.Context,
	path string,
) (puzzleInspection, bool, error) {
	if err := ctx.Err(); err != nil {
		return puzzleInspection{}, false, err
	}
	file, err := os.Open(path)
	if err != nil {
		return puzzleInspection{}, false, err
	}
	defer file.Close()

	scanner := newLinearFENLineScanner(contextReader{ctx: ctx, reader: file})
	lineNumber := int64(0)
	for scanner.Scan() {
		lineNumber++
		if err := ctx.Err(); err != nil {
			return puzzleInspection{}, false, err
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !looksLikeLinearFENRecord(line) {
			continue
		}
		return puzzleInspection{
			SourceID:       path,
			SourceIDOrigin: SourceIDPath,
		}, true, nil
	}
	if err := scanner.Err(); err != nil {
		return puzzleInspection{}, false, fmt.Errorf(
			"inspect linear FEN/UCI line %d: %w",
			lineNumber+1,
			err,
		)
	}
	return puzzleInspection{}, false, nil
}

func looksLikeLinearFENRecord(line string) bool {
	fields := strings.Fields(line)
	if len(fields) < 7 || !looksLikeFENFields(fields[:6]) {
		return false
	}
	moves := fields[6:]
	if len(moves) > 1 {
		if _, err := strconv.Atoi(moves[len(moves)-1]); err == nil {
			moves = moves[:len(moves)-1]
		}
	}
	if len(moves) == 0 {
		return false
	}
	for _, move := range moves {
		if !looksLikeUCIMove(move) {
			return false
		}
	}
	return true
}

func looksLikeFENFields(fields []string) bool {
	if len(fields) != 6 || (fields[1] != "w" && fields[1] != "b") {
		return false
	}
	ranks := strings.Split(fields[0], "/")
	if len(ranks) != 8 {
		return false
	}
	for _, rank := range ranks {
		squares := 0
		for _, piece := range rank {
			switch {
			case piece >= '1' && piece <= '8':
				squares += int(piece - '0')
			case strings.ContainsRune("prnbqkPRNBQK", piece):
				squares++
			default:
				return false
			}
		}
		if squares != 8 {
			return false
		}
	}
	if fields[2] != "-" {
		for _, right := range fields[2] {
			if !strings.ContainsRune("KQkq", right) {
				return false
			}
		}
	}
	if fields[3] != "-" && (len(fields[3]) != 2 || fields[3][0] < 'a' ||
		fields[3][0] > 'h' || fields[3][1] < '1' || fields[3][1] > '8') {
		return false
	}
	if _, err := strconv.ParseUint(fields[4], 10, 64); err != nil {
		return false
	}
	if _, err := strconv.ParseUint(fields[5], 10, 64); err != nil {
		return false
	}
	return true
}

func looksLikeUCIMove(move string) bool {
	if len(move) != 4 && len(move) != 5 {
		return false
	}
	if move[0] < 'a' || move[0] > 'h' || move[1] < '1' || move[1] > '8' ||
		move[2] < 'a' || move[2] > 'h' || move[3] < '1' || move[3] > '8' {
		return false
	}
	return len(move) == 4 || strings.ContainsRune("qrbnQRBN", rune(move[4]))
}

func (a linearFENAdapter) NewDecoder(
	reader io.Reader,
	_ puzzleInspection,
) (PuzzleDecoder, error) {
	if reader == nil {
		return nil, errors.New("linear FEN/UCI reader is required")
	}
	return &linearFENDecoder{
		rules:   a.rules,
		scanner: newLinearFENLineScanner(reader),
	}, nil
}

func newLinearFENLineScanner(reader io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), maxLinearFENLineBytes)
	return scanner
}

type linearFENDecoder struct {
	rules      chessrules.Rules
	scanner    *bufio.Scanner
	lineNumber int64
	closed     bool
}

func (d *linearFENDecoder) Next(ctx context.Context) (DecodedRecord, error) {
	if err := ctx.Err(); err != nil {
		return DecodedRecord{}, err
	}
	if d.closed {
		return DecodedRecord{}, io.EOF
	}
	for d.scanner.Scan() {
		d.lineNumber++
		if err := ctx.Err(); err != nil {
			return DecodedRecord{}, err
		}
		line := strings.TrimSpace(d.scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		puzzle, err := d.decodeLine(line)
		if err != nil {
			return linearFENRejection(d.lineNumber, err), nil
		}
		return DecodedRecord{Puzzle: &puzzle}, nil
	}
	if err := d.scanner.Err(); err != nil {
		return DecodedRecord{}, fmt.Errorf(
			"scan linear FEN/UCI line %d: %w",
			d.lineNumber+1,
			err,
		)
	}
	return DecodedRecord{}, io.EOF
}

func (d *linearFENDecoder) decodeLine(line string) (TrainingPuzzle, error) {
	core, metadata, err := parseLinearFENLine(d.rules, line)
	if err != nil {
		return TrainingPuzzle{}, err
	}
	return TrainingPuzzle{
		Core: core,
		Occurrence: PuzzleOccurrence{
			ExternalID: strconv.FormatInt(d.lineNumber, 10),
			SourceFEN:  core.DisplayedFEN,
			Metadata:   metadata,
			Ordinal:    d.lineNumber,
		},
	}, nil
}

func parseLinearFENLine(
	rules chessrules.Rules,
	line string,
) (PuzzleCore, map[string]any, error) {
	fields := strings.Fields(line)
	if len(fields) < 7 {
		return PuzzleCore{}, nil, errors.New(
			"line must contain six FEN fields and at least one UCI move",
		)
	}

	fen := strings.Join(fields[:6], " ")
	normalizedFEN, err := normalizeFEN(rules, fen)
	if err != nil {
		return PuzzleCore{}, nil, fmt.Errorf("normalize FEN: %w", err)
	}
	moves := fields[6:]
	metadata := map[string]any{}
	if difficulty, err := strconv.Atoi(moves[len(moves)-1]); err == nil {
		metadata["sourceDifficulty"] = difficulty
		moves = moves[:len(moves)-1]
	}
	if len(moves) == 0 {
		return PuzzleCore{}, nil, errors.New("at least one UCI move is required")
	}

	solver, err := solverFromFEN(normalizedFEN)
	if err != nil {
		return PuzzleCore{}, nil, err
	}
	solution, err := linearSolution(moves)
	if err != nil {
		return PuzzleCore{}, nil, fmt.Errorf("validate linear UCI move line: %w", err)
	}
	core, err := finalizeCore(rules, normalizedFEN, solver, solution)
	if err != nil {
		return PuzzleCore{}, nil, fmt.Errorf("validate linear UCI move line: %w", err)
	}
	return core, metadata, nil
}

func (d *linearFENDecoder) Close() error {
	d.closed = true
	return nil
}

func linearFENRejection(ordinal int64, err error) DecodedRecord {
	rejection := Rejection{Ordinal: ordinal, Reason: err.Error()}
	return DecodedRecord{Rejection: &rejection}
}
