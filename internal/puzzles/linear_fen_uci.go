package puzzles

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"chess-trainer/internal/chessrules"
)

const (
	maxLinearFENLineBytes     = 1 << 20
	maxLinearFENMetadataBytes = 64 * 1024
)

type linearFENAdapter struct {
	rules chessrules.Rules
}

func NewLinearFENAdapter(rules chessrules.Rules) PuzzleAdapter {
	return linearFENAdapter{rules: rules}
}

func (linearFENAdapter) Format() ImportFormat {
	return FormatLinearFENUCI
}

func (a linearFENAdapter) Inspect(
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

	scanner := newLinearFENLineScanner(contextReader{ctx: ctx, reader: file})
	lineNumber := int64(0)
	for scanner.Scan() {
		lineNumber++
		if err := ctx.Err(); err != nil {
			return ImportInspection{}, false, err
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if _, _, err := parseLinearFENLine(a.rules, line); err != nil {
			return ImportInspection{}, false, nil
		}
		return ImportInspection{
			SourceID:       path,
			SourceIDOrigin: SourceIDPath,
		}, true, nil
	}
	if err := scanner.Err(); err != nil {
		return ImportInspection{}, false, fmt.Errorf(
			"inspect linear FEN/UCI line %d: %w",
			lineNumber+1,
			err,
		)
	}
	return ImportInspection{}, false, nil
}

func (a linearFENAdapter) NewDecoder(
	reader io.Reader,
	inspection ImportInspection,
) (PuzzleDecoder, error) {
	if reader == nil {
		return nil, errors.New("linear FEN/UCI reader is required")
	}
	sourceID := strings.TrimSpace(inspection.SourceID)
	if sourceID == "" {
		return nil, errors.New("linear FEN/UCI source ID is required")
	}
	return &linearFENDecoder{
		rules:    a.rules,
		sourceID: sourceID,
		scanner:  newLinearFENLineScanner(reader),
	}, nil
}

func newLinearFENLineScanner(reader io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), maxLinearFENLineBytes)
	return scanner
}

type linearFENDecoder struct {
	rules      chessrules.Rules
	sourceID   string
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
			SourceID:   d.sourceID,
			SourceKind: string(FormatLinearFENUCI),
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
	encodedMetadata, err := json.Marshal(metadata)
	if err != nil {
		return PuzzleCore{}, nil, fmt.Errorf("serialize linear FEN/UCI metadata: %w", err)
	}
	if len(encodedMetadata) > maxLinearFENMetadataBytes {
		return PuzzleCore{}, nil, fmt.Errorf(
			"metadata exceeds maximum of %d bytes",
			maxLinearFENMetadataBytes,
		)
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
