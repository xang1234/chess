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
	"chess-trainer/internal/importing"
)

const (
	canonicalJSONSchema           = "chess-trainer-puzzles/v1"
	maxCanonicalJSONPuzzleBytes   = 2 * 1024 * 1024
	maxCanonicalJSONMetadataBytes = 64 * 1024
)

type canonicalJSONAdapter struct {
	rules chessrules.Rules
}

func NewCanonicalJSONAdapter(rules chessrules.Rules) PuzzleAdapter {
	return canonicalJSONAdapter{rules: rules}
}

func (canonicalJSONAdapter) Descriptor() ImportFormatDescriptor {
	return ImportFormatDescriptor{
		Format: FormatCanonicalJSON, Label: "Canonical JSON",
		CanonicalExtension: ".json", FileFilterDescription: "JSON collection",
	}
}

func (a canonicalJSONAdapter) Inspect(
	ctx context.Context,
	path string,
) (importing.Inspection, bool, error) {
	if err := ctx.Err(); err != nil {
		return importing.Inspection{}, false, err
	}
	file, err := os.Open(path)
	if err != nil {
		return importing.Inspection{}, false, err
	}
	defer file.Close()

	descriptor, err := inspectCanonicalJSONDocument(
		contextReader{ctx: ctx, reader: file},
	)
	if err != nil {
		return importing.Inspection{}, false, fmt.Errorf("inspect canonical JSON: %w", err)
	}
	inspection := importing.Inspection{
		SourceID:       descriptor.ID,
		SourceIDOrigin: SourceIDEmbedded,
		SourceName:     descriptor.Name,
		URL:            descriptor.URL,
		Attribution:    descriptor.Attribution,
	}
	if inspection.SourceID == "" {
		inspection.SourceID = path
		inspection.SourceIDOrigin = SourceIDPath
	}
	return inspection, true, nil
}

func inspectCanonicalJSONDocument(reader io.Reader) (canonicalSource, error) {
	stream := newCanonicalJSONDocumentStream(reader)
	seen := make(map[string]struct{}, len(canonicalTopLevelFields))
	var source canonicalSource
	for {
		key, done, err := stream.nextField()
		if err != nil {
			return canonicalSource{}, err
		}
		if done {
			break
		}
		if err := acceptCanonicalKey(seen, canonicalTopLevelFields, "top-level", key); err != nil {
			return canonicalSource{}, err
		}
		switch key {
		case "schema":
			raw, oversized, err := stream.readFieldValue(maxCanonicalJSONPuzzleBytes)
			if err != nil {
				return canonicalSource{}, fmt.Errorf("decode schema: %w", err)
			}
			if oversized {
				return canonicalSource{}, errors.New("canonical JSON schema value is too large")
			}
			var schema string
			if err := json.Unmarshal(raw, &schema); err != nil {
				return canonicalSource{}, fmt.Errorf("decode schema: %w", err)
			}
			if schema != canonicalJSONSchema {
				return canonicalSource{}, fmt.Errorf(
					"unsupported canonical JSON schema %q; want %q",
					schema,
					canonicalJSONSchema,
				)
			}
		case "source":
			raw, oversized, err := stream.readFieldValue(maxCanonicalJSONPuzzleBytes)
			if err != nil {
				return canonicalSource{}, fmt.Errorf("decode source: %w", err)
			}
			if oversized {
				return canonicalSource{}, errors.New("canonical JSON source descriptor is too large")
			}
			source, err = decodeCanonicalSource(raw)
			if err != nil {
				return canonicalSource{}, err
			}
		case "puzzles":
			if err := stream.beginArray(); err != nil {
				return canonicalSource{}, err
			}
			for {
				_, _, done, err := stream.nextArrayValue(0)
				if err != nil {
					return canonicalSource{}, fmt.Errorf("decode puzzle framing: %w", err)
				}
				if done {
					break
				}
			}
		}
	}
	if _, present := seen["schema"]; !present {
		return canonicalSource{}, errors.New("canonical JSON schema is required")
	}
	if _, present := seen["puzzles"]; !present {
		return canonicalSource{}, errors.New("canonical JSON puzzles array is required")
	}
	if err := stream.requireEOF(); err != nil {
		return canonicalSource{}, err
	}
	return source, nil
}

func (a canonicalJSONAdapter) NewDecoder(
	reader io.Reader,
	inspection importing.Inspection,
) (PuzzleDecoder, error) {
	if reader == nil {
		return nil, errors.New("canonical JSON reader is required")
	}
	if strings.TrimSpace(inspection.SourceID) == "" {
		return nil, errors.New("canonical JSON source ID is required")
	}
	return &canonicalJSONDecoder{
		rules:      a.rules,
		inspection: inspection,
		stream:     newCanonicalJSONDocumentStream(reader),
		seen:       make(map[string]struct{}, len(canonicalTopLevelFields)),
	}, nil
}

type canonicalJSONDecoder struct {
	rules      chessrules.Rules
	inspection importing.Inspection
	stream     *canonicalJSONDocumentStream
	seen       map[string]struct{}
	inPuzzles  bool
	finished   bool
	closed     bool
	ordinal    int64
}

func (d *canonicalJSONDecoder) Next(ctx context.Context) (DecodedRecord, error) {
	if err := ctx.Err(); err != nil {
		return DecodedRecord{}, err
	}
	if d.closed || d.finished {
		return DecodedRecord{}, io.EOF
	}
	for {
		if err := ctx.Err(); err != nil {
			return DecodedRecord{}, err
		}
		if d.inPuzzles {
			raw, oversized, done, err := d.stream.nextArrayValue(maxCanonicalJSONPuzzleBytes)
			if err != nil {
				return DecodedRecord{}, fmt.Errorf(
					"decode canonical JSON puzzle %d framing: %w",
					d.ordinal+1,
					err,
				)
			}
			if done {
				d.inPuzzles = false
				continue
			}
			d.ordinal++
			if oversized {
				return canonicalJSONRejection(d.ordinal, fmt.Errorf(
					"canonical JSON puzzle exceeds maximum of %d bytes",
					maxCanonicalJSONPuzzleBytes,
				)), nil
			}
			return d.decodeRecord(raw), nil
		}

		key, done, err := d.stream.nextField()
		if err != nil {
			return DecodedRecord{}, err
		}
		if !done {
			if err := acceptCanonicalKey(d.seen, canonicalTopLevelFields, "top-level", key); err != nil {
				return DecodedRecord{}, err
			}
			switch key {
			case "schema":
				if err := d.decodeSchema(); err != nil {
					return DecodedRecord{}, err
				}
			case "source":
				if err := d.decodeSource(); err != nil {
					return DecodedRecord{}, err
				}
			case "puzzles":
				if err := d.stream.beginArray(); err != nil {
					return DecodedRecord{}, err
				}
				d.inPuzzles = true
			}
			continue
		}

		if _, present := d.seen["schema"]; !present {
			return DecodedRecord{}, errors.New("canonical JSON schema is required")
		}
		if _, present := d.seen["puzzles"]; !present {
			return DecodedRecord{}, errors.New("canonical JSON puzzles array is required")
		}
		if _, sourceSeen := d.seen["source"]; !sourceSeen {
			if d.inspection.SourceIDOrigin != SourceIDPath ||
				d.inspection.SourceName != "" ||
				d.inspection.URL != "" ||
				d.inspection.Attribution != "" {
				return DecodedRecord{}, errors.New(
					"canonical JSON source changed after inspection: source object is missing",
				)
			}
		}
		if err := d.stream.requireEOF(); err != nil {
			return DecodedRecord{}, err
		}
		d.finished = true
		return DecodedRecord{}, io.EOF
	}
}

func (d *canonicalJSONDecoder) decodeSchema() error {
	raw, oversized, err := d.stream.readFieldValue(maxCanonicalJSONPuzzleBytes)
	if err != nil {
		return fmt.Errorf("decode schema: %w", err)
	}
	if oversized {
		return errors.New("canonical JSON schema value is too large")
	}
	var schema string
	if err := json.Unmarshal(raw, &schema); err != nil {
		return fmt.Errorf("decode schema: %w", err)
	}
	if schema != canonicalJSONSchema {
		return fmt.Errorf(
			"canonical JSON schema changed after inspection: got %q, want %q",
			schema,
			canonicalJSONSchema,
		)
	}
	return nil
}

func (d *canonicalJSONDecoder) decodeSource() error {
	raw, oversized, err := d.stream.readFieldValue(maxCanonicalJSONPuzzleBytes)
	if err != nil {
		return fmt.Errorf("decode source: %w", err)
	}
	if oversized {
		return errors.New("canonical JSON source descriptor is too large")
	}
	source, err := decodeCanonicalSource(raw)
	if err != nil {
		return err
	}
	if source.ID == "" {
		if d.inspection.SourceIDOrigin != SourceIDPath {
			return errors.New(
				"canonical JSON source changed after inspection: embedded source ID is missing",
			)
		}
	} else if d.inspection.SourceIDOrigin != SourceIDEmbedded || source.ID != d.inspection.SourceID {
		return fmt.Errorf(
			"canonical JSON source changed after inspection: got ID %q, want %q",
			source.ID,
			d.inspection.SourceID,
		)
	}
	if source.Name != d.inspection.SourceName ||
		source.URL != d.inspection.URL ||
		source.Attribution != d.inspection.Attribution {
		return errors.New("canonical JSON source metadata changed after inspection")
	}
	return nil
}

func (d *canonicalJSONDecoder) decodeRecord(raw json.RawMessage) DecodedRecord {
	if len(raw) > maxCanonicalJSONPuzzleBytes {
		return canonicalJSONRejection(d.ordinal, fmt.Errorf(
			"canonical JSON puzzle exceeds maximum of %d bytes",
			maxCanonicalJSONPuzzleBytes,
		))
	}
	puzzle, err := d.normalizePuzzle(raw)
	if err != nil {
		return canonicalJSONRejection(d.ordinal, err)
	}
	return DecodedRecord{Puzzle: &puzzle}
}

func (d *canonicalJSONDecoder) normalizePuzzle(raw json.RawMessage) (TrainingPuzzle, error) {
	fields, seen, err := decodeCanonicalPuzzle(raw)
	if err != nil {
		return TrainingPuzzle{}, err
	}
	if fields.DisplayedFEN == nil {
		return TrainingPuzzle{}, errors.New("displayedFen is required and must be a string")
	}
	if fields.Solver == nil {
		return TrainingPuzzle{}, errors.New("solver is required and must be a string")
	}
	if _, present := seen["solution"]; !present {
		return TrainingPuzzle{}, errors.New("solution is required and must be an array")
	}

	var solver domain.Color
	switch *fields.Solver {
	case string(domain.White):
		solver = domain.White
	case string(domain.Black):
		solver = domain.Black
	default:
		return TrainingPuzzle{}, fmt.Errorf(
			"solver must be %q or %q",
			domain.White,
			domain.Black,
		)
	}
	if fields.Rating != nil && (*fields.Rating < 100 || *fields.Rating > 4000) {
		return TrainingPuzzle{}, fmt.Errorf("rating %d is outside 100-4000", *fields.Rating)
	}
	if fields.Popularity != nil && *fields.Popularity < 0 {
		return TrainingPuzzle{}, errors.New("popularity must be non-negative")
	}
	if fields.PlayCount != nil && *fields.PlayCount < 0 {
		return TrainingPuzzle{}, errors.New("playCount must be non-negative")
	}

	core, err := finalizeCore(d.rules, *fields.DisplayedFEN, solver, fields.Solution)
	if err != nil {
		return TrainingPuzzle{}, fmt.Errorf("validate canonical puzzle: %w", err)
	}

	sourceFEN, preludeUCI, err := d.presentation(fields, seen, core.DisplayedFEN)
	if err != nil {
		return TrainingPuzzle{}, err
	}
	metadata, err := decodeCanonicalMetadata(fields.Metadata, seen)
	if err != nil {
		return TrainingPuzzle{}, err
	}
	externalID := ""
	if fields.ID != nil {
		externalID = strings.TrimSpace(*fields.ID)
	}
	if externalID == "" {
		externalID = strconv.FormatInt(d.ordinal, 10)
	}
	url := d.inspection.URL
	if fields.URL != nil {
		url = strings.TrimSpace(*fields.URL)
	}
	attribution := d.inspection.Attribution
	if fields.Attribution != nil {
		attribution = strings.TrimSpace(*fields.Attribution)
	}
	var themes []string
	if fields.Themes != nil {
		themes = domain.NormalizeThemes(*fields.Themes)
	}

	return TrainingPuzzle{
		Core: core,
		Occurrence: PuzzleOccurrence{
			ExternalID:  externalID,
			SourceFEN:   sourceFEN,
			PreludeUCI:  preludeUCI,
			Rating:      fields.Rating,
			Popularity:  fields.Popularity,
			PlayCount:   fields.PlayCount,
			URL:         url,
			Attribution: attribution,
			Metadata:    metadata,
			Themes:      themes,
			Ordinal:     d.ordinal,
		},
	}, nil
}

func (d *canonicalJSONDecoder) presentation(
	fields canonicalPuzzle,
	seen map[string]struct{},
	displayedFEN string,
) (string, string, error) {
	_, hasSourceFEN := seen["sourceFen"]
	_, hasPrelude := seen["preludeUci"]
	if hasSourceFEN != hasPrelude {
		return "", "", errors.New("sourceFen and preludeUci must appear together")
	}
	if !hasSourceFEN {
		return "", "", nil
	}
	if fields.SourceFEN == nil || fields.PreludeUCI == nil {
		return "", "", errors.New("sourceFen and preludeUci must be non-null strings")
	}
	normalizedSourceFEN, err := normalizeFEN(d.rules, *fields.SourceFEN)
	if err != nil {
		return "", "", fmt.Errorf("normalize sourceFen: %w", err)
	}
	preludeUCI := strings.ToLower(strings.TrimSpace(*fields.PreludeUCI))
	afterPrelude, err := d.rules.ApplyUCI(normalizedSourceFEN, preludeUCI)
	if err != nil {
		return "", "", fmt.Errorf("apply preludeUci %q: %w", preludeUCI, err)
	}
	if afterPrelude != displayedFEN {
		return "", "", fmt.Errorf(
			"preludeUci result does not match displayedFen: got %q, want %q",
			afterPrelude,
			displayedFEN,
		)
	}
	return normalizedSourceFEN, preludeUCI, nil
}

func (d *canonicalJSONDecoder) Close() error {
	d.closed = true
	return nil
}

var _ PuzzleAdapter = canonicalJSONAdapter{}
var _ PuzzleDecoder = (*canonicalJSONDecoder)(nil)
