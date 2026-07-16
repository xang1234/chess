package puzzles

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/domain"
	"chess-trainer/internal/storage"

	"github.com/klauspost/compress/zstd"
)

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

type Progress struct {
	RowsRead  int64 `json:"rowsRead"`
	BytesRead int64 `json:"bytesRead"`
}

type ProgressSink func(Progress)

type LichessImporter struct {
	Catalog          CatalogWriter
	Rules            chessrules.Rules
	CatalogDirectory string
	AvailableBytes   func(string) (uint64, error)
}

const abandonImportTimeout = 5 * time.Second

type countingReader struct {
	reader io.Reader
	read   int64
}

func (r *countingReader) Read(buffer []byte) (int, error) {
	count, err := r.reader.Read(buffer)
	r.read += int64(count)
	return count, err
}

func (i LichessImporter) Import(
	ctx context.Context,
	sourceID string,
	path string,
	progress ProgressSink,
) (ImportReport, error) {
	info, err := os.Stat(path)
	if err != nil {
		return ImportReport{}, err
	}
	availableBytes := i.AvailableBytes
	if availableBytes == nil {
		availableBytes = storage.AvailableBytes
	}
	if strings.TrimSpace(i.CatalogDirectory) == "" {
		return ImportReport{}, errors.New("puzzle catalogue directory is required")
	}
	available, err := availableBytes(i.CatalogDirectory)
	if err != nil {
		return ImportReport{}, err
	}
	required := storage.RequiredImportBytes(info.Size())
	if available < required {
		return ImportReport{}, fmt.Errorf(
			"not enough free disk space: have %d bytes, need %d bytes",
			available,
			required,
		)
	}

	file, err := os.Open(path)
	if err != nil {
		return ImportReport{}, err
	}
	defer file.Close()

	generation, err := i.Catalog.BeginImport(ctx, Source{
		ID:        sourceID,
		Kind:      "lichess",
		Path:      path,
		StartedAt: time.Now(),
	})
	if err != nil {
		return ImportReport{}, err
	}
	generation = newOrderedGenerationImport(ctx, generation)
	sealed := false
	abandon := func(cause error) (ImportReport, error) {
		if !sealed {
			cleanupContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), abandonImportTimeout)
			defer cancel()
			if abandonErr := generation.Abandon(cleanupContext); abandonErr != nil {
				cause = errors.Join(cause, fmt.Errorf("abandon import: %w", abandonErr))
			}
		}
		return ImportReport{}, cause
	}

	hash := sha256.New()
	counter := &countingReader{reader: file}
	compressed := io.TeeReader(counter, hash)
	decoder, err := zstd.NewReader(
		compressed,
		zstd.WithDecoderConcurrency(1),
		zstd.WithDecoderLowmem(true),
	)
	if err != nil {
		return abandon(err)
	}
	defer decoder.Close()

	reader := csv.NewReader(decoder)
	reader.ReuseRecord = true
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return abandon(fmt.Errorf("read Lichess header: %w", err))
	}
	columns, err := lichessColumnIndexes(header)
	if err != nil {
		return abandon(err)
	}

	var rowsRead int64
	for {
		if err := ctx.Err(); err != nil {
			return abandon(err)
		}
		record, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		rowsRead++
		ordinal := rowsRead + 1
		if readErr != nil {
			if len(record) == 0 {
				return abandon(fmt.Errorf("read CSV row %d: %w", ordinal, readErr))
			}
			generation.Reject(Rejection{Ordinal: ordinal, Reason: readErr.Error()})
			continue
		}

		puzzle, err := i.normalizeRecord(sourceID, ordinal, record, columns)
		if err != nil {
			generation.Reject(Rejection{Ordinal: ordinal, Reason: err.Error()})
		} else if err := generation.Add(ctx, puzzle); err != nil {
			return abandon(err)
		}
		if progress != nil && rowsRead%10_000 == 0 {
			progress(Progress{RowsRead: rowsRead, BytesRead: counter.read})
		}
	}

	decoder.Close()
	if _, err := io.Copy(io.Discard, compressed); err != nil {
		return abandon(fmt.Errorf("finish source checksum: %w", err))
	}
	if progress != nil {
		progress(Progress{RowsRead: rowsRead, BytesRead: counter.read})
	}
	report, err := generation.Seal(ctx, hex.EncodeToString(hash.Sum(nil)))
	if err != nil {
		return abandon(err)
	}
	sealed = true
	if err := generation.Activate(ctx); err != nil {
		return report, err
	}
	return report, nil
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

func (i LichessImporter) normalizeRecord(
	sourceID string,
	ordinal int64,
	record []string,
	columns map[string]int,
) (TrainingPuzzle, error) {
	value := func(name string) (string, error) {
		index := columns[name]
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

	displayedFEN, err := i.Rules.ApplyUCILine(sourceFEN, moves)
	if err != nil {
		return TrainingPuzzle{}, fmt.Errorf("validate move line: %w", err)
	}
	solver, err := solverFromFEN(displayedFEN)
	if err != nil {
		return TrainingPuzzle{}, err
	}
	rating, err := nullableInteger(record, columns, "Rating")
	if err != nil {
		return TrainingPuzzle{}, err
	}
	ratingDeviation, err := nullableInteger(record, columns, "RatingDeviation")
	if err != nil {
		return TrainingPuzzle{}, err
	}
	popularity, err := nullableInteger(record, columns, "Popularity")
	if err != nil {
		return TrainingPuzzle{}, err
	}
	playCount, err := nullableInteger(record, columns, "NbPlays")
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

	core := PuzzleCore{
		DisplayedFEN:  displayedFEN,
		Solver:        solver,
		Solution:      moveLine(moves[1:]),
		SolutionPlies: len(moves) - 1,
	}
	core.Fingerprint, err = CoreFingerprint(core)
	if err != nil {
		return TrainingPuzzle{}, err
	}
	return TrainingPuzzle{
		Core: core,
		Occurrence: PuzzleOccurrence{
			SourceID:    sourceID,
			SourceKind:  "lichess",
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

func moveLine(moves []string) []domain.MoveNode {
	if len(moves) == 0 {
		return nil
	}
	return []domain.MoveNode{{
		UCI:      strings.ToLower(moves[0]),
		Children: moveLine(moves[1:]),
	}}
}
