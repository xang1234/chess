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

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(buffer)
}

type CollectionImporter struct {
	Catalog          CatalogWriter
	Reader           CatalogReader
	Adapters         []PuzzleAdapter
	CatalogDirectory string
	AvailableBytes   func(string) (uint64, error)
}

func (i *CollectionImporter) Supports(format ImportFormat) bool {
	if i == nil {
		return false
	}
	for _, adapter := range i.Adapters {
		if adapter != nil && adapter.Format() == format {
			return true
		}
	}
	return false
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
	_, inspection, err := i.inspectRegistry(ctx, normalizedPath)
	return inspection, err
}

func (i CollectionImporter) inspectRegistry(
	ctx context.Context,
	normalizedPath string,
) (PuzzleAdapter, ImportInspection, error) {
	if err := ctx.Err(); err != nil {
		return nil, ImportInspection{}, err
	}
	matches := make([]adapterInspection, 0, len(i.Adapters))
	var inspectionErrors []error
	var contextErrors []error
	for _, adapter := range i.Adapters {
		if adapter == nil {
			inspectionErrors = append(inspectionErrors, errors.New("puzzle adapter is nil"))
			continue
		}
		inspection, matched, inspectErr := adapter.Inspect(ctx, normalizedPath)
		if inspectErr != nil {
			wrapped := fmt.Errorf("inspect %s puzzle source: %w", adapter.Format(), inspectErr)
			if errors.Is(inspectErr, context.Canceled) || errors.Is(inspectErr, context.DeadlineExceeded) {
				contextErrors = append(contextErrors, wrapped)
			} else {
				inspectionErrors = append(inspectionErrors, wrapped)
			}
			continue
		}
		if matched {
			matches = append(matches, adapterInspection{adapter: adapter, inspection: inspection})
		}
	}
	if err := ctx.Err(); err != nil {
		contextErrors = append(contextErrors, err)
	}
	if err := errors.Join(contextErrors...); err != nil {
		return nil, ImportInspection{}, err
	}
	if len(matches) == 0 {
		if err := errors.Join(inspectionErrors...); err != nil {
			return nil, ImportInspection{}, err
		}
		return nil, ImportInspection{}, errors.New("unsupported puzzle import format")
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
			return nil, ImportInspection{}, fmt.Errorf(
				"ambiguous puzzle import format: content matches %s",
				strings.Join(formats, ", "),
			)
		}
		selected = extensionMatches[0]
	}

	inspection, err := i.completeInspection(
		ctx, normalizedPath, selected.adapter, selected.inspection,
	)
	if err != nil {
		return nil, ImportInspection{}, err
	}
	return selected.adapter, inspection, nil
}

func (i CollectionImporter) Import(
	ctx context.Context,
	expected ImportInspection,
	progress ProgressSink,
) (ImportReport, error) {
	emit := importProgressEmitter(progress)
	emit(ImportDetecting, 0, 0, 0)

	adapter, inspection, err := i.revalidateInspection(ctx, expected)
	if err != nil {
		return ImportReport{}, err
	}
	return i.importResolved(ctx, adapter, inspection, emit)
}

func importProgressEmitter(progress ProgressSink) func(ImportPhase, int64, int64, int64) {
	return func(phase ImportPhase, rowsRead, bytesRead, totalBytes int64) {
		if progress != nil {
			progress(Progress{
				Phase: phase, RowsRead: rowsRead, BytesRead: bytesRead, TotalBytes: totalBytes,
			})
		}
	}
}

func (i CollectionImporter) importResolved(
	ctx context.Context,
	adapter PuzzleAdapter,
	inspection ImportInspection,
	emit func(ImportPhase, int64, int64, int64),
) (ImportReport, error) {
	normalizedPath := inspection.Path
	format := inspection.Format
	sourceID := inspection.SourceID

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
	if _, err := io.Copy(io.Discard, contextReader{ctx: ctx, reader: raw}); err != nil {
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

func (i CollectionImporter) revalidateInspection(
	ctx context.Context,
	expected ImportInspection,
) (PuzzleAdapter, ImportInspection, error) {
	if err := ctx.Err(); err != nil {
		return nil, ImportInspection{}, err
	}
	normalizedPath, err := normalizeImportPath(expected.Path)
	if err != nil {
		return nil, ImportInspection{}, err
	}
	if expected.Path != normalizedPath {
		return nil, ImportInspection{}, inspectionChangedError("path", normalizedPath, expected.Path)
	}
	adapter, err := i.adapterForFormat(expected.Format)
	if err != nil {
		return nil, ImportInspection{}, err
	}
	current, matched, err := adapter.Inspect(ctx, normalizedPath)
	if err != nil {
		return nil, ImportInspection{}, fmt.Errorf(
			"reinspect %s puzzle source: %w",
			expected.Format,
			err,
		)
	}
	if !matched {
		return nil, ImportInspection{}, fmt.Errorf(
			"puzzle import no longer matches inspected format %q",
			expected.Format,
		)
	}
	current, err = normalizeAdapterInspection(normalizedPath, adapter, current)
	if err != nil {
		return nil, ImportInspection{}, err
	}
	if err := compareImportInspection(current, expected); err != nil {
		return nil, ImportInspection{}, err
	}
	return adapter, current, nil
}

func (i CollectionImporter) adapterForFormat(format ImportFormat) (PuzzleAdapter, error) {
	var selected PuzzleAdapter
	for _, adapter := range i.Adapters {
		if adapter == nil || adapter.Format() != format {
			continue
		}
		if selected != nil {
			return nil, fmt.Errorf("multiple puzzle adapters configured for format %q", format)
		}
		selected = adapter
	}
	if selected == nil {
		return nil, fmt.Errorf("puzzle adapter for format %q is not configured", format)
	}
	return selected, nil
}

func compareImportInspection(current, expected ImportInspection) error {
	fields := []struct {
		name          string
		current, want string
	}{
		{name: "path", current: current.Path, want: expected.Path},
		{name: "filename", current: current.Filename, want: expected.Filename},
		{name: "format", current: string(current.Format), want: string(expected.Format)},
		{name: "source ID", current: current.SourceID, want: expected.SourceID},
		{
			name: "source ID origin", current: string(current.SourceIDOrigin),
			want: string(expected.SourceIDOrigin),
		},
		{name: "source name", current: current.SourceName, want: expected.SourceName},
		{name: "source URL", current: current.URL, want: expected.URL},
		{name: "source attribution", current: current.Attribution, want: expected.Attribution},
	}
	for _, field := range fields {
		if field.current != field.want {
			return inspectionChangedError(field.name, field.current, field.want)
		}
	}
	return nil
}

func inspectionChangedError(field, current, expected string) error {
	return fmt.Errorf(
		"puzzle import %s changed after inspection: got %q, want %q",
		field,
		current,
		expected,
	)
}

func (i CollectionImporter) completeInspection(
	ctx context.Context,
	normalizedPath string,
	adapter PuzzleAdapter,
	inspection ImportInspection,
) (ImportInspection, error) {
	if err := ctx.Err(); err != nil {
		return ImportInspection{}, err
	}
	inspection, err := normalizeAdapterInspection(normalizedPath, adapter, inspection)
	if err != nil {
		return ImportInspection{}, err
	}
	if i.Reader == nil {
		if err := ctx.Err(); err != nil {
			return ImportInspection{}, err
		}
		return inspection, nil
	}
	summaries, err := i.Reader.ActiveSourceSummaries(ctx)
	if err != nil {
		return ImportInspection{}, err
	}
	if err := ctx.Err(); err != nil {
		return ImportInspection{}, err
	}
	for _, summary := range summaries {
		if summary.SourceID == inspection.SourceID {
			inspection.ReplacesExisting = true
			break
		}
	}
	if err := ctx.Err(); err != nil {
		return ImportInspection{}, err
	}
	return inspection, nil
}

func normalizeAdapterInspection(
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
