package puzzles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"chess-trainer/internal/storage"
)

type ImportFormat string

const (
	FormatLichess       ImportFormat = "lichess"
	FormatTacticalPGN   ImportFormat = "tactical-pgn"
	FormatCanonicalJSON ImportFormat = "canonical-json"
	FormatLucasFNS      ImportFormat = "lucas-fns"
	FormatLinearFENUCI  ImportFormat = "linear-fen-uci"
)

type SourceIDOrigin string

const (
	SourceIDFixed    SourceIDOrigin = "fixed"
	SourceIDEmbedded SourceIDOrigin = "embedded"
	SourceIDPath     SourceIDOrigin = "path"
)

type ImportInspection struct {
	Path             string         `json:"path"`
	Filename         string         `json:"filename"`
	Format           ImportFormat   `json:"format"`
	SourceID         string         `json:"sourceId"`
	SourceIDOrigin   SourceIDOrigin `json:"sourceIdOrigin"`
	SourceName       string         `json:"sourceName,omitempty"`
	URL              string         `json:"url,omitempty"`
	Attribution      string         `json:"attribution,omitempty"`
	ReplacesExisting bool           `json:"replacesExisting"`
}

type DecodedRecord struct {
	Puzzle    *TrainingPuzzle
	Rejection *Rejection
}

type PuzzleDecoder interface {
	Next(context.Context) (DecodedRecord, error)
	Close() error
}

type PuzzleAdapter interface {
	Format() ImportFormat
	Inspect(context.Context, string) (ImportInspection, bool, error)
	NewDecoder(io.Reader, ImportInspection) (PuzzleDecoder, error)
}

type ImportPhase string

const (
	ImportDetecting  ImportPhase = "detecting"
	ImportParsing    ImportPhase = "parsing"
	ImportSealing    ImportPhase = "sealing"
	ImportActivating ImportPhase = "activating"
)

type Progress struct {
	Phase      ImportPhase `json:"phase"`
	RowsRead   int64       `json:"rowsRead"`
	BytesRead  int64       `json:"bytesRead"`
	TotalBytes int64       `json:"totalBytes"`
}

type ProgressSink func(Progress)

var ErrNoValidPuzzles = errors.New("puzzle import contains no valid puzzles")

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

type CollectionImporter struct {
	Catalog          CatalogWriter
	Reader           CatalogReader
	Adapters         []PuzzleAdapter
	CatalogDirectory string
	AvailableBytes   func(string) (uint64, error)
}

type FormatImporter struct {
	Collection *CollectionImporter
	Format     ImportFormat
}

func (i FormatImporter) Import(
	ctx context.Context,
	sourceID string,
	path string,
	progress ProgressSink,
) (ImportReport, error) {
	if i.Collection == nil {
		return ImportReport{}, errors.New("puzzle collection importer is required")
	}
	return i.Collection.ImportFormat(ctx, i.Format, sourceID, path, progress)
}

type adapterInspection struct {
	adapter    PuzzleAdapter
	inspection ImportInspection
}

func (i CollectionImporter) Inspect(ctx context.Context, path string) (ImportInspection, error) {
	normalizedPath, err := normalizeImportPath(path)
	if err != nil {
		return ImportInspection{}, err
	}

	matches := make([]adapterInspection, 0, len(i.Adapters))
	var inspectionErrors []error
	for _, adapter := range i.Adapters {
		if adapter == nil {
			inspectionErrors = append(inspectionErrors, errors.New("puzzle adapter is nil"))
			continue
		}
		inspection, matched, inspectErr := adapter.Inspect(ctx, normalizedPath)
		if inspectErr != nil {
			inspectionErrors = append(
				inspectionErrors,
				fmt.Errorf("inspect %s puzzle source: %w", adapter.Format(), inspectErr),
			)
			continue
		}
		if matched {
			matches = append(matches, adapterInspection{adapter: adapter, inspection: inspection})
		}
	}
	if len(matches) == 0 {
		if err := errors.Join(inspectionErrors...); err != nil {
			return ImportInspection{}, err
		}
		return ImportInspection{}, errors.New("unsupported puzzle import format")
	}

	selected := matches[0]
	if len(matches) > 1 {
		extensionMatches := make([]adapterInspection, 0, len(matches))
		for _, match := range matches {
			if importFormatMatchesExtension(match.adapter.Format(), normalizedPath) {
				extensionMatches = append(extensionMatches, match)
			}
		}
		if len(extensionMatches) != 1 {
			formats := make([]string, 0, len(matches))
			for _, match := range matches {
				formats = append(formats, string(match.adapter.Format()))
			}
			return ImportInspection{}, fmt.Errorf(
				"ambiguous puzzle import format: content matches %s",
				strings.Join(formats, ", "),
			)
		}
		selected = extensionMatches[0]
	}

	return i.completeInspection(ctx, normalizedPath, selected.adapter, selected.inspection)
}

func (i CollectionImporter) ImportFormat(
	ctx context.Context,
	format ImportFormat,
	sourceID string,
	path string,
	progress ProgressSink,
) (ImportReport, error) {
	emit := func(phase ImportPhase, rowsRead, bytesRead, totalBytes int64) {
		if progress != nil {
			progress(Progress{
				Phase: phase, RowsRead: rowsRead, BytesRead: bytesRead, TotalBytes: totalBytes,
			})
		}
	}
	emit(ImportDetecting, 0, 0, 0)

	normalizedPath, err := normalizeImportPath(path)
	if err != nil {
		return ImportReport{}, err
	}
	adapter, inspection, err := i.inspectFormat(ctx, format, normalizedPath)
	if err != nil {
		return ImportReport{}, err
	}
	if inspection.Path != normalizedPath {
		return ImportReport{}, fmt.Errorf(
			"puzzle import path changed after inspection: got %q, want %q",
			inspection.Path,
			normalizedPath,
		)
	}
	if sourceID != inspection.SourceID {
		return ImportReport{}, fmt.Errorf(
			"puzzle import source ID changed after inspection: got %q, want %q",
			inspection.SourceID,
			sourceID,
		)
	}

	info, err := os.Stat(normalizedPath)
	if err != nil {
		return ImportReport{}, err
	}
	totalBytes := info.Size()
	emit(ImportDetecting, 0, 0, totalBytes)
	if strings.TrimSpace(i.CatalogDirectory) == "" {
		return ImportReport{}, errors.New("puzzle catalogue directory is required")
	}
	availableBytes := i.AvailableBytes
	if availableBytes == nil {
		availableBytes = storage.AvailableBytes
	}
	available, err := availableBytes(i.CatalogDirectory)
	if err != nil {
		return ImportReport{}, err
	}
	required := storage.RequiredImportBytes(totalBytes)
	if available < required {
		return ImportReport{}, fmt.Errorf(
			"not enough free disk space: have %d bytes, need %d bytes",
			available,
			required,
		)
	}
	if i.Catalog == nil {
		return ImportReport{}, errors.New("puzzle catalogue writer is required")
	}

	file, err := os.Open(normalizedPath)
	if err != nil {
		return ImportReport{}, err
	}
	defer file.Close()

	generation, err := i.Catalog.BeginImport(ctx, Source{
		ID: sourceID, Kind: string(format), Path: normalizedPath, StartedAt: time.Now(),
	})
	if err != nil {
		return ImportReport{}, err
	}
	if generation == nil {
		return ImportReport{}, errors.New("puzzle catalogue returned a nil generation import")
	}
	ordered := newOrderedGenerationImport(ctx, generation)
	sealed := false

	hash := sha256.New()
	counter := &countingReader{reader: file}
	raw := io.TeeReader(counter, hash)
	decoder, decoderErr := adapter.NewDecoder(raw, inspection)
	decoderClosed := false
	closeDecoder := func() error {
		if decoder == nil || decoderClosed {
			return nil
		}
		decoderClosed = true
		return decoder.Close()
	}
	abandon := func(cause error) (ImportReport, error) {
		if closeErr := closeDecoder(); closeErr != nil {
			cause = errors.Join(cause, fmt.Errorf("close puzzle decoder: %w", closeErr))
		}
		if !sealed {
			cleanupContext, cancel := context.WithTimeout(
				context.WithoutCancel(ctx),
				abandonImportTimeout,
			)
			defer cancel()
			if abandonErr := ordered.Abandon(cleanupContext); abandonErr != nil {
				cause = errors.Join(cause, fmt.Errorf("abandon import: %w", abandonErr))
			}
		}
		return ImportReport{}, cause
	}
	if decoderErr != nil {
		return abandon(decoderErr)
	}
	if decoder == nil {
		return abandon(errors.New("puzzle adapter returned a nil decoder"))
	}

	var rowsRead, validPuzzles int64
	emit(ImportParsing, rowsRead, counter.read, totalBytes)
	for {
		if err := ctx.Err(); err != nil {
			return abandon(err)
		}
		record, nextErr := decoder.Next(ctx)
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return abandon(nextErr)
		}
		rowsRead++
		if err := ctx.Err(); err != nil {
			return abandon(err)
		}
		if (record.Puzzle == nil) == (record.Rejection == nil) {
			return abandon(errors.New("decoded record must contain exactly one puzzle or rejection"))
		}
		if record.Puzzle != nil {
			if err := ordered.Add(ctx, *record.Puzzle); err != nil {
				return abandon(err)
			}
			validPuzzles++
		} else {
			ordered.Reject(*record.Rejection)
		}
		if rowsRead%10_000 == 0 {
			emit(ImportParsing, rowsRead, counter.read, totalBytes)
		}
	}
	if err := ctx.Err(); err != nil {
		return abandon(err)
	}
	if err := closeDecoder(); err != nil {
		return abandon(fmt.Errorf("close puzzle decoder: %w", err))
	}
	if _, err := io.Copy(io.Discard, raw); err != nil {
		return abandon(fmt.Errorf("finish source checksum: %w", err))
	}
	emit(ImportParsing, rowsRead, counter.read, totalBytes)
	if err := ctx.Err(); err != nil {
		return abandon(err)
	}
	if validPuzzles == 0 {
		return abandon(ErrNoValidPuzzles)
	}

	emit(ImportSealing, rowsRead, counter.read, totalBytes)
	report, err := ordered.Seal(ctx, hex.EncodeToString(hash.Sum(nil)))
	if err != nil {
		return abandon(err)
	}
	sealed = true
	emit(ImportActivating, rowsRead, counter.read, totalBytes)
	if err := ordered.Activate(ctx); err != nil {
		return report, err
	}
	return report, nil
}

func (i CollectionImporter) inspectFormat(
	ctx context.Context,
	format ImportFormat,
	normalizedPath string,
) (PuzzleAdapter, ImportInspection, error) {
	var selected PuzzleAdapter
	for _, adapter := range i.Adapters {
		if adapter != nil && adapter.Format() == format {
			if selected != nil {
				return nil, ImportInspection{}, fmt.Errorf(
					"multiple puzzle adapters are configured for format %q",
					format,
				)
			}
			selected = adapter
		}
	}
	if selected == nil {
		return nil, ImportInspection{}, fmt.Errorf(
			"puzzle adapter for format %q is not configured",
			format,
		)
	}
	inspection, matched, err := selected.Inspect(ctx, normalizedPath)
	if err != nil {
		return nil, ImportInspection{}, err
	}
	if !matched {
		return nil, ImportInspection{}, fmt.Errorf(
			"puzzle source no longer matches requested format %q",
			format,
		)
	}
	inspection, err = i.completeInspection(ctx, normalizedPath, selected, inspection)
	if err != nil {
		return nil, ImportInspection{}, err
	}
	return selected, inspection, nil
}

func (i CollectionImporter) completeInspection(
	ctx context.Context,
	normalizedPath string,
	adapter PuzzleAdapter,
	inspection ImportInspection,
) (ImportInspection, error) {
	inspection.Path = normalizedPath
	inspection.Filename = filepath.Base(normalizedPath)
	inspection.Format = adapter.Format()
	if inspection.SourceIDOrigin == SourceIDPath {
		inspection.SourceID = normalizedPath
	}
	if strings.TrimSpace(inspection.SourceID) == "" {
		return ImportInspection{}, fmt.Errorf(
			"%s puzzle inspection returned an empty source ID",
			adapter.Format(),
		)
	}
	if i.Reader == nil {
		return inspection, nil
	}
	summaries, err := i.Reader.ActiveSourceSummaries(ctx)
	if err != nil {
		return ImportInspection{}, err
	}
	for _, summary := range summaries {
		if summary.SourceID == inspection.SourceID {
			inspection.ReplacesExisting = true
			break
		}
	}
	return inspection, nil
}

func normalizeImportPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("puzzle import path is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	normalized := filepath.Clean(absolute)
	if resolved, err := filepath.EvalSymlinks(normalized); err == nil {
		normalized = filepath.Clean(resolved)
	}
	return normalized, nil
}

func importFormatMatchesExtension(format ImportFormat, path string) bool {
	extension := strings.ToLower(filepath.Ext(path))
	switch format {
	case FormatLichess:
		return extension == ".zst"
	case FormatTacticalPGN:
		return extension == ".pgn"
	case FormatCanonicalJSON:
		return extension == ".json"
	case FormatLucasFNS:
		return extension == ".fns"
	case FormatLinearFENUCI:
		return extension == ".txt"
	default:
		return false
	}
}

var _ interface {
	Import(context.Context, string, string, ProgressSink) (ImportReport, error)
} = FormatImporter{}
