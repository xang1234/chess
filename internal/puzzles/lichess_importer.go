package puzzles

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/domain"

	"github.com/klauspost/compress/zstd"
)

var lichessZstandardMagic = [4]byte{0x28, 0xb5, 0x2f, 0xfd}

var lichessColumns = []string{
	"PuzzleId",
	"FEN",
	"Moves",
	"Rating",
	"RatingDeviation",
	"Popularity",
	"NbPlays",
	"Themes",
	"GameUrl",
	"OpeningTags",
}

type lichessAdapter struct {
	rules chessrules.Rules
}

func NewLichessAdapter(rules chessrules.Rules) PuzzleAdapter {
	return lichessAdapter{rules: rules}
}

func (lichessAdapter) Descriptor() ImportFormatDescriptor {
	return ImportFormatDescriptor{
		Format: FormatLichess, Label: "Lichess",
		CanonicalExtension: ".zst", FileFilterDescription: "Zstandard archive",
	}
}

func (lichessAdapter) Inspect(
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

	var magic [len(lichessZstandardMagic)]byte
	if _, err := io.ReadFull(contextReader{ctx: ctx, reader: file}, magic[:]); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return puzzleInspection{}, false, nil
		}
		return puzzleInspection{}, false, err
	}
	if magic != lichessZstandardMagic {
		return puzzleInspection{}, false, nil
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return puzzleInspection{}, false, err
	}

	decoder, err := newLichessZstandardReader(contextReader{ctx: ctx, reader: file})
	if err != nil {
		return puzzleInspection{}, false, fmt.Errorf("unsupported Lichess content: %w", err)
	}
	defer decoder.Close()
	reader := newLichessCSVReader(decoder)
	header, err := reader.Read()
	if err != nil {
		return puzzleInspection{}, false, fmt.Errorf("unsupported Lichess content: read header: %w", err)
	}
	if _, err := lichessColumnIndexes(header); err != nil {
		return puzzleInspection{}, false, fmt.Errorf("unsupported Lichess content: %w", err)
	}
	return puzzleInspection{
		SourceID:       "lichess",
		SourceIDOrigin: SourceIDFixed,
	}, true, nil
}

func (a lichessAdapter) NewDecoder(
	reader io.Reader,
	_ puzzleInspection,
) (PuzzleDecoder, error) {
	decoder, err := newLichessZstandardReader(reader)
	if err != nil {
		return nil, err
	}
	csvReader := newLichessCSVReader(decoder)
	header, err := csvReader.Read()
	if err != nil {
		decoder.Close()
		return nil, fmt.Errorf("read Lichess header: %w", err)
	}
	columns, err := lichessColumnIndexes(header)
	if err != nil {
		decoder.Close()
		return nil, err
	}
	return &lichessDecoder{
		rules:   a.rules,
		decoder: decoder,
		reader:  csvReader,
		columns: columns,
	}, nil
}

func newLichessZstandardReader(reader io.Reader) (*zstd.Decoder, error) {
	return zstd.NewReader(
		reader,
		zstd.WithDecoderConcurrency(1),
		zstd.WithDecoderLowmem(true),
	)
}

func newLichessCSVReader(reader io.Reader) *csv.Reader {
	csvReader := csv.NewReader(reader)
	csvReader.ReuseRecord = true
	csvReader.FieldsPerRecord = -1
	return csvReader
}

type lichessDecoder struct {
	rules    chessrules.Rules
	decoder  *zstd.Decoder
	reader   *csv.Reader
	columns  map[string]int
	rowsRead int64
	closed   bool
}

func (d *lichessDecoder) Next(ctx context.Context) (DecodedRecord, error) {
	if err := ctx.Err(); err != nil {
		return DecodedRecord{}, err
	}
	record, readErr := d.reader.Read()
	if errors.Is(readErr, io.EOF) {
		return DecodedRecord{}, io.EOF
	}
	d.rowsRead++
	ordinal := d.rowsRead + 1
	if readErr != nil {
		if len(record) == 0 {
			return DecodedRecord{}, fmt.Errorf("read CSV row %d: %w", ordinal, readErr)
		}
		rejection := Rejection{Ordinal: ordinal, Reason: readErr.Error()}
		return DecodedRecord{Rejection: &rejection}, nil
	}

	puzzle, err := d.normalizeRecord(ordinal, record)
	if err != nil {
		rejection := Rejection{Ordinal: ordinal, Reason: err.Error()}
		return DecodedRecord{Rejection: &rejection}, nil
	}
	return DecodedRecord{Puzzle: &puzzle}, nil
}

func (d *lichessDecoder) Close() error {
	if d.closed {
		return nil
	}
	d.closed = true
	d.decoder.Close()
	return nil
}

func lichessColumnIndexes(header []string) (map[string]int, error) {
	columns := make(map[string]int, len(header))
	for index, name := range header {
		name = strings.TrimSpace(name)
		if _, duplicate := columns[name]; duplicate {
			return nil, fmt.Errorf("duplicate Lichess column %q", name)
		}
		columns[name] = index
	}
	for _, required := range lichessColumns {
		if _, ok := columns[required]; !ok {
			return nil, fmt.Errorf("missing required Lichess column %q", required)
		}
	}
	return columns, nil
}

func (d *lichessDecoder) normalizeRecord(
	ordinal int64,
	record []string,
) (TrainingPuzzle, error) {
	value := func(name string) (string, error) {
		index := d.columns[name]
		if index >= len(record) {
			return "", fmt.Errorf("row has no %s field", name)
		}
		return strings.TrimSpace(record[index]), nil
	}
	puzzleID, err := value("PuzzleId")
	if err != nil || puzzleID == "" {
		return TrainingPuzzle{}, errors.New("PuzzleId is required")
	}
	sourceFEN, err := value("FEN")
	if err != nil {
		return TrainingPuzzle{}, err
	}
	movesField, err := value("Moves")
	if err != nil {
		return TrainingPuzzle{}, err
	}
	moves := strings.Fields(movesField)
	if len(moves) < 2 {
		return TrainingPuzzle{}, errors.New("Moves must contain a setup move and at least one solution move")
	}

	displayedFEN, err := d.rules.ApplyUCI(sourceFEN, moves[0])
	if err != nil {
		return TrainingPuzzle{}, fmt.Errorf("validate move line: %w", err)
	}
	solver, err := solverFromFEN(displayedFEN)
	if err != nil {
		return TrainingPuzzle{}, err
	}
	solution, err := linearSolution(moves[1:])
	if err != nil {
		return TrainingPuzzle{}, fmt.Errorf("validate move line: %w", err)
	}
	core, err := finalizeCore(d.rules, displayedFEN, solver, solution)
	if err != nil {
		return TrainingPuzzle{}, fmt.Errorf("validate move line: %w", err)
	}
	rating, err := nullableInteger(record, d.columns, "Rating")
	if err != nil {
		return TrainingPuzzle{}, err
	}
	ratingDeviation, err := nullableInteger(record, d.columns, "RatingDeviation")
	if err != nil {
		return TrainingPuzzle{}, err
	}
	popularity, err := nullableInteger(record, d.columns, "Popularity")
	if err != nil {
		return TrainingPuzzle{}, err
	}
	playCount, err := nullableInteger(record, d.columns, "NbPlays")
	if err != nil {
		return TrainingPuzzle{}, err
	}
	themesField, err := value("Themes")
	if err != nil {
		return TrainingPuzzle{}, err
	}
	themes := domain.NormalizeThemes(strings.Fields(themesField))
	gameURL, err := value("GameUrl")
	if err != nil {
		return TrainingPuzzle{}, err
	}
	openingTags, err := value("OpeningTags")
	if err != nil {
		return TrainingPuzzle{}, err
	}
	metadata := map[string]any{}
	if ratingDeviation != nil {
		metadata["ratingDeviation"] = *ratingDeviation
	}
	if openingTags != "" {
		metadata["openingTags"] = strings.Fields(openingTags)
	}

	return TrainingPuzzle{
		Core: core,
		Occurrence: PuzzleOccurrence{
			ExternalID:  puzzleID,
			SourceFEN:   sourceFEN,
			PreludeUCI:  strings.ToLower(moves[0]),
			Rating:      rating,
			Popularity:  popularity,
			PlayCount:   playCount,
			URL:         gameURL,
			Attribution: "Lichess puzzle database (CC0)",
			Metadata:    metadata,
			Themes:      themes,
			Ordinal:     ordinal,
		},
	}, nil
}

func nullableInteger(record []string, columns map[string]int, name string) (*int, error) {
	index := columns[name]
	if index >= len(record) {
		return nil, fmt.Errorf("row has no %s field", name)
	}
	field := strings.TrimSpace(record[index])
	if field == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(field)
	if err != nil {
		return nil, fmt.Errorf("parse %s %q: %w", name, field, err)
	}
	return &value, nil
}
