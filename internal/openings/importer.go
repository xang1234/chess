package openings

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"chess-trainer/internal/importing"
)

const (
	FormatCoursePack   importing.Format = "coursepack"
	MaxCoursePackBytes int64            = 32 << 20
)

type courseReplacer interface {
	Replace(context.Context, CompiledCourse, string, string) (ReplaceResult, error)
	ActiveGenerationID(context.Context, string) (string, error)
}

type Importer struct {
	Catalog courseReplacer
	Rules   RulesPort
}

func NewImporter(catalog courseReplacer, rules RulesPort) *Importer {
	return &Importer{Catalog: catalog, Rules: rules}
}

func ValidateCoursePackFile(
	ctx context.Context,
	path string,
	rules RulesPort,
) (CompiledCourse, error) {
	normalizedPath, err := importing.NormalizePath(path, "course import")
	if err != nil {
		return CompiledCourse{}, err
	}
	compiled, _, _, err := (&Importer{Rules: rules}).readAndCompile(ctx, normalizedPath)
	return compiled, err
}

func (i *Importer) Supports(format importing.Format) bool {
	return i != nil && format == FormatCoursePack
}

func (i *Importer) Inspect(ctx context.Context, path string) (importing.Inspection, error) {
	normalizedPath, err := importing.NormalizePath(path, "course import")
	if err != nil {
		return importing.Inspection{}, err
	}
	compiled, _, _, err := i.readAndCompile(ctx, normalizedPath)
	if err != nil {
		return importing.Inspection{}, err
	}
	inspection := courseInspection(normalizedPath, compiled.Pack)
	if i.Catalog == nil {
		return inspection, errors.New("course catalog is required")
	}
	if _, err := i.Catalog.ActiveGenerationID(ctx, compiled.Pack.CourseID); err == nil {
		inspection.ReplacesExisting = true
	} else if !errors.Is(err, sql.ErrNoRows) {
		return importing.Inspection{}, fmt.Errorf("inspect active course: %w", err)
	}
	return inspection, nil
}

func (i *Importer) Import(
	ctx context.Context,
	expected importing.Inspection,
	progress importing.ProgressSink,
) (importing.Report, error) {
	emitCourseProgress(progress, importing.PhaseDetecting, 0, 0, 0)
	if i == nil || i.Catalog == nil {
		return importing.Report{}, errors.New("course importer catalog is required")
	}
	normalizedPath, err := importing.NormalizePath(expected.Path, "course import")
	if err != nil {
		return importing.Report{}, err
	}
	if normalizedPath != expected.Path {
		current := expected
		current.Path = normalizedPath
		return importing.Report{}, importing.CompareInspection(current, expected, "course import")
	}
	info, err := os.Stat(normalizedPath)
	if err != nil {
		return importing.Report{}, fmt.Errorf("stat course pack: %w", err)
	}
	emitCourseProgress(progress, importing.PhaseDetecting, 0, 0, info.Size())

	compiled, checksum, bytesRead, err := i.readAndCompile(ctx, normalizedPath)
	if err != nil {
		return importing.Report{}, err
	}
	current := courseInspection(normalizedPath, compiled.Pack)
	if err := importing.CompareInspection(current, expected, "course import"); err != nil {
		return importing.Report{}, err
	}

	rowsRead := int64(0)
	emitCourseProgress(progress, importing.PhaseParsing, rowsRead, bytesRead, bytesRead)
	for _, count := range []int{
		len(compiled.Pack.Chapters),
		len(compiled.Pack.Positions),
		len(compiled.Pack.Moves),
		len(compiled.Pack.Notes),
		len(compiled.Pack.Lessons),
		len(compiled.Pack.Prompts),
	} {
		rowsRead += int64(count)
		emitCourseProgress(progress, importing.PhaseParsing, rowsRead, bytesRead, bytesRead)
	}
	if err := ctx.Err(); err != nil {
		return importing.Report{}, err
	}
	emitCourseProgress(progress, importing.PhaseSealing, rowsRead, bytesRead, bytesRead)
	if _, err := i.Catalog.Replace(ctx, compiled, normalizedPath, checksum); err != nil {
		return importing.Report{}, err
	}
	emitCourseProgress(progress, importing.PhaseActivating, rowsRead, bytesRead, bytesRead)
	return importing.Report{Accepted: 1, Counts: StructuralCounts(compiled)}, nil
}

func (i *Importer) readAndCompile(
	ctx context.Context,
	path string,
) (CompiledCourse, string, int64, error) {
	if i == nil || i.Rules == nil {
		return CompiledCourse{}, "", 0, errors.New("course chess rules are required")
	}
	if err := ctx.Err(); err != nil {
		return CompiledCourse{}, "", 0, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return CompiledCourse{}, "", 0, fmt.Errorf("stat course pack: %w", err)
	}
	if info.Size() > MaxCoursePackBytes {
		return CompiledCourse{}, "", 0, coursePackTooLarge(info.Size())
	}
	file, err := os.Open(path)
	if err != nil {
		return CompiledCourse{}, "", 0, fmt.Errorf("open course pack: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	counter := &courseCountingReader{reader: io.TeeReader(
		io.LimitReader(courseContextReader{ctx: ctx, reader: file}, MaxCoursePackBytes+1),
		hash,
	)}
	pack, decodeErr := DecodeCoursePack(counter)
	if counter.read > MaxCoursePackBytes {
		return CompiledCourse{}, "", counter.read, coursePackTooLarge(counter.read)
	}
	if decodeErr != nil {
		return CompiledCourse{}, "", counter.read, decodeErr
	}
	if err := ctx.Err(); err != nil {
		return CompiledCourse{}, "", counter.read, err
	}
	compiled, err := Compile(pack, i.Rules)
	if err != nil {
		return compiled, "", counter.read, err
	}
	return compiled, hex.EncodeToString(hash.Sum(nil)), counter.read, nil
}

func StructuralCounts(compiled CompiledCourse) map[string]int64 {
	variations := map[string]struct{}{}
	for _, move := range compiled.Moves {
		name := strings.TrimSpace(move.VariationName)
		if name != "" {
			variations[name] = struct{}{}
		}
	}
	var warnings int64
	for _, note := range compiled.Notes {
		if note.Kind == "warning" {
			warnings++
		}
	}
	var activities int64
	for _, lesson := range compiled.Lessons {
		activities += int64(len(lesson.Activities))
	}
	return map[string]int64{
		"chapters":    int64(len(compiled.Chapters)),
		"positions":   int64(len(compiled.Positions)),
		"moves":       int64(len(compiled.Moves)),
		"variations":  int64(len(variations)),
		"notes":       int64(len(compiled.Notes)),
		"lessons":     int64(len(compiled.Lessons)),
		"lessonEdges": int64(len(compiled.Pack.LessonEdges)),
		"activities":  activities,
		"prompts":     int64(len(compiled.Prompts)),
		"warnings":    warnings,
	}
}

func courseInspection(path string, pack CoursePack) importing.Inspection {
	return importing.Inspection{
		Path:           path,
		Filename:       filepath.Base(path),
		Format:         FormatCoursePack,
		FormatLabel:    "Opening course",
		SourceID:       pack.CourseID,
		SourceIDOrigin: importing.SourceIDEmbedded,
		SourceName:     pack.Title,
		Attribution:    strings.TrimSpace(pack.Source.Title + ", " + pack.Source.Edition),
	}
}

func emitCourseProgress(
	sink importing.ProgressSink,
	phase importing.Phase,
	rowsRead int64,
	bytesRead int64,
	totalBytes int64,
) {
	if sink != nil {
		sink(importing.Progress{
			Phase: phase, RowsRead: rowsRead, BytesRead: bytesRead, TotalBytes: totalBytes,
		})
	}
}

func coursePackTooLarge(size int64) error {
	return fmt.Errorf("course pack is %d bytes; maximum size is 32 MiB", size)
}

type courseCountingReader struct {
	reader io.Reader
	read   int64
}

func (r *courseCountingReader) Read(buffer []byte) (int, error) {
	count, err := r.reader.Read(buffer)
	r.read += int64(count)
	return count, err
}

type courseContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r courseContextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(buffer)
}
