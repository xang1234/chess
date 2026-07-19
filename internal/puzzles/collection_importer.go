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

	"chess-trainer/internal/importing"
	"chess-trainer/internal/storage"
)

type ImportFormat = importing.Format

const (
	FormatLichess       ImportFormat = "lichess"
	FormatTacticalPGN   ImportFormat = "tactical-pgn"
	FormatCanonicalJSON ImportFormat = "canonical-json"
	FormatLucasFNS      ImportFormat = "lucas-fns"
	FormatLinearFENUCI  ImportFormat = "linear-fen-uci"
)

type SourceIDOrigin = importing.SourceIDOrigin

const (
	SourceIDFixed    = importing.SourceIDFixed
	SourceIDEmbedded = importing.SourceIDEmbedded
	SourceIDPath     = importing.SourceIDPath
)

type puzzleInspection = importing.Inspection

type DecodedRecord struct {
	Puzzle    *TrainingPuzzle
	Rejection *Rejection
}

type PuzzleDecoder interface {
	Next(context.Context) (DecodedRecord, error)
	Close() error
}

type ImportFormatDescriptor struct {
	Format                ImportFormat
	Label                 string
	CanonicalExtension    string
	FileFilterDescription string
}

type PuzzleAdapter interface {
	Descriptor() ImportFormatDescriptor
	Inspect(context.Context, string) (puzzleInspection, bool, error)
	NewDecoder(io.Reader, puzzleInspection) (PuzzleDecoder, error)
}

type ImportPhase = importing.Phase

const (
	ImportDetecting  = importing.PhaseDetecting
	ImportParsing    = importing.PhaseParsing
	ImportSealing    = importing.PhaseSealing
	ImportActivating = importing.PhaseActivating
)

type puzzleProgress = importing.Progress
type puzzleProgressSink = importing.ProgressSink

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
	descriptors, err := i.FormatDescriptors()
	if err != nil {
		return false
	}
	for _, descriptor := range descriptors {
		if descriptor.Format == format {
			return true
		}
	}
	return false
}

func (i *CollectionImporter) FormatDescriptors() ([]ImportFormatDescriptor, error) {
	if i == nil {
		return nil, errors.New("puzzle collection importer is required")
	}
	descriptors := make([]ImportFormatDescriptor, 0, len(i.Adapters))
	for index, adapter := range i.Adapters {
		if adapter == nil {
			return nil, fmt.Errorf("puzzle adapter %d is nil", index)
		}
		descriptor := adapter.Descriptor()
		if err := validateImportFormatDescriptor(descriptor); err != nil {
			return nil, fmt.Errorf("puzzle adapter %d descriptor: %w", index, err)
		}
		descriptors = append(descriptors, descriptor)
	}
	return descriptors, nil
}

func validateImportFormatDescriptor(descriptor ImportFormatDescriptor) error {
	if strings.TrimSpace(string(descriptor.Format)) == "" {
		return errors.New("format is required")
	}
	if strings.TrimSpace(descriptor.Label) == "" {
		return errors.New("label is required")
	}
	extension := strings.TrimSpace(descriptor.CanonicalExtension)
	if extension == "" || extension != descriptor.CanonicalExtension ||
		!strings.HasPrefix(extension, ".") || strings.Contains(extension, "*") {
		return errors.New("canonical extension must begin with '.' and contain no wildcard")
	}
	if strings.TrimSpace(descriptor.FileFilterDescription) == "" {
		return errors.New("file filter description is required")
	}
	return nil
}

type adapterInspection struct {
	adapter    PuzzleAdapter
	descriptor ImportFormatDescriptor
	inspection puzzleInspection
}

func (i CollectionImporter) Inspect(ctx context.Context, path string) (puzzleInspection, error) {
	normalizedPath, err := importing.NormalizePath(path, "puzzle import")
	if err != nil {
		return puzzleInspection{}, err
	}
	_, inspection, err := i.inspectRegistry(ctx, normalizedPath)
	return inspection, err
}

func (i CollectionImporter) inspectRegistry(
	ctx context.Context,
	normalizedPath string,
) (PuzzleAdapter, puzzleInspection, error) {
	if err := ctx.Err(); err != nil {
		return nil, puzzleInspection{}, err
	}
	descriptors, err := (&i).FormatDescriptors()
	if err != nil {
		return nil, puzzleInspection{}, err
	}
	matches := make([]adapterInspection, 0, len(i.Adapters))
	var inspectionErrors []error
	var contextErrors []error
	for index, adapter := range i.Adapters {
		descriptor := descriptors[index]
		inspection, matched, inspectErr := adapter.Inspect(ctx, normalizedPath)
		if inspectErr != nil {
			wrapped := fmt.Errorf("inspect %s puzzle source: %w", descriptor.Format, inspectErr)
			if errors.Is(inspectErr, context.Canceled) || errors.Is(inspectErr, context.DeadlineExceeded) {
				contextErrors = append(contextErrors, wrapped)
			} else {
				inspectionErrors = append(inspectionErrors, wrapped)
			}
			continue
		}
		if matched {
			matches = append(matches, adapterInspection{
				adapter: adapter, descriptor: descriptor, inspection: inspection,
			})
		}
	}
	if err := ctx.Err(); err != nil {
		contextErrors = append(contextErrors, err)
	}
	if err := errors.Join(contextErrors...); err != nil {
		return nil, puzzleInspection{}, err
	}
	if len(matches) == 0 {
		if err := errors.Join(inspectionErrors...); err != nil {
			return nil, puzzleInspection{}, err
		}
		return nil, puzzleInspection{}, errors.New("unsupported puzzle import format")
	}

	selected := matches[0]
	if len(matches) > 1 {
		extensionMatches := make([]adapterInspection, 0, len(matches))
		for _, match := range matches {
			if descriptorMatchesExtension(match.descriptor, normalizedPath) {
				extensionMatches = append(extensionMatches, match)
			}
		}
		if len(extensionMatches) != 1 {
			formats := make([]string, 0, len(matches))
			for _, match := range matches {
				formats = append(formats, string(match.descriptor.Format))
			}
			return nil, puzzleInspection{}, fmt.Errorf(
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
		return nil, puzzleInspection{}, err
	}
	return selected.adapter, inspection, nil
}

func (i CollectionImporter) Import(
	ctx context.Context,
	expected puzzleInspection,
	progress puzzleProgressSink,
) (ImportReport, error) {
	emit := importProgressEmitter(progress)
	emit(ImportDetecting, 0, 0, 0)

	adapter, inspection, err := i.revalidateInspection(ctx, expected)
	if err != nil {
		return ImportReport{}, err
	}
	return i.importResolved(ctx, adapter, inspection, emit)
}

func importProgressEmitter(progress puzzleProgressSink) func(ImportPhase, int64, int64, int64) {
	return func(phase ImportPhase, rowsRead, bytesRead, totalBytes int64) {
		if progress != nil {
			progress(puzzleProgress{
				Phase: phase, RowsRead: rowsRead, BytesRead: bytesRead, TotalBytes: totalBytes,
			})
		}
	}
}

func (i CollectionImporter) importResolved(
	ctx context.Context,
	adapter PuzzleAdapter,
	inspection puzzleInspection,
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
			puzzle := *record.Puzzle
			puzzle.Occurrence.SourceID = sourceID
			puzzle.Occurrence.SourceKind = string(format)
			if err := ordered.Add(ctx, puzzle); err != nil {
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
	expected puzzleInspection,
) (PuzzleAdapter, puzzleInspection, error) {
	if err := ctx.Err(); err != nil {
		return nil, puzzleInspection{}, err
	}
	normalizedPath, err := importing.NormalizePath(expected.Path, "puzzle import")
	if err != nil {
		return nil, puzzleInspection{}, err
	}
	if expected.Path != normalizedPath {
		current := expected
		current.Path = normalizedPath
		return nil, puzzleInspection{}, importing.CompareInspection(current, expected, "puzzle import")
	}
	adapter, err := i.adapterForFormat(expected.Format)
	if err != nil {
		return nil, puzzleInspection{}, err
	}
	current, matched, err := adapter.Inspect(ctx, normalizedPath)
	if err != nil {
		return nil, puzzleInspection{}, fmt.Errorf(
			"reinspect %s puzzle source: %w",
			expected.Format,
			err,
		)
	}
	if !matched {
		return nil, puzzleInspection{}, fmt.Errorf(
			"puzzle import no longer matches inspected format %q",
			expected.Format,
		)
	}
	current, err = normalizeAdapterInspection(normalizedPath, adapter, current)
	if err != nil {
		return nil, puzzleInspection{}, err
	}
	if err := importing.CompareInspection(current, expected, "puzzle import"); err != nil {
		return nil, puzzleInspection{}, err
	}
	return adapter, current, nil
}

func (i CollectionImporter) adapterForFormat(format ImportFormat) (PuzzleAdapter, error) {
	descriptors, err := (&i).FormatDescriptors()
	if err != nil {
		return nil, err
	}
	var selected PuzzleAdapter
	for index, adapter := range i.Adapters {
		if descriptors[index].Format != format {
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

func (i CollectionImporter) completeInspection(
	ctx context.Context,
	normalizedPath string,
	adapter PuzzleAdapter,
	inspection puzzleInspection,
) (puzzleInspection, error) {
	if err := ctx.Err(); err != nil {
		return puzzleInspection{}, err
	}
	inspection, err := normalizeAdapterInspection(normalizedPath, adapter, inspection)
	if err != nil {
		return puzzleInspection{}, err
	}
	if i.Reader == nil {
		if err := ctx.Err(); err != nil {
			return puzzleInspection{}, err
		}
		return inspection, nil
	}
	summaries, err := i.Reader.ActiveSourceSummaries(ctx)
	if err != nil {
		return puzzleInspection{}, err
	}
	if err := ctx.Err(); err != nil {
		return puzzleInspection{}, err
	}
	for _, summary := range summaries {
		if summary.SourceID == inspection.SourceID {
			inspection.ReplacesExisting = true
			break
		}
	}
	if err := ctx.Err(); err != nil {
		return puzzleInspection{}, err
	}
	return inspection, nil
}

func normalizeAdapterInspection(
	normalizedPath string,
	adapter PuzzleAdapter,
	inspection puzzleInspection,
) (puzzleInspection, error) {
	descriptor := adapter.Descriptor()
	if err := validateImportFormatDescriptor(descriptor); err != nil {
		return puzzleInspection{}, fmt.Errorf("puzzle adapter descriptor: %w", err)
	}
	inspection.Path = normalizedPath
	inspection.Filename = filepath.Base(normalizedPath)
	inspection.Format = descriptor.Format
	inspection.FormatLabel = descriptor.Label
	if inspection.SourceIDOrigin == SourceIDPath {
		inspection.SourceID = normalizedPath
	}
	if strings.TrimSpace(inspection.SourceID) == "" {
		return puzzleInspection{}, fmt.Errorf(
			"%s puzzle inspection returned an empty source ID",
			descriptor.Format,
		)
	}
	return inspection, nil
}

func descriptorMatchesExtension(descriptor ImportFormatDescriptor, path string) bool {
	return strings.EqualFold(filepath.Ext(path), descriptor.CanonicalExtension)
}
