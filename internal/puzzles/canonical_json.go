package puzzles

import (
	"bytes"
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
)

const (
	canonicalJSONSchema           = "chess-trainer-puzzles/v1"
	maxCanonicalJSONPuzzleBytes   = 2 * 1024 * 1024
	maxCanonicalJSONMetadataBytes = 64 * 1024
)

var (
	canonicalTopLevelFields = map[string]struct{}{
		"schema": {}, "source": {}, "puzzles": {},
	}
	canonicalSourceFields = map[string]struct{}{
		"id": {}, "name": {}, "url": {}, "attribution": {},
	}
	canonicalPuzzleFields = map[string]struct{}{
		"id": {}, "sourceFen": {}, "preludeUci": {}, "displayedFen": {},
		"solver": {}, "solution": {}, "rating": {}, "themes": {},
		"popularity": {}, "playCount": {}, "url": {}, "attribution": {},
		"metadata": {},
	}
	canonicalMoveFields = map[string]struct{}{
		"uci": {}, "children": {},
	}
)

type canonicalJSONAdapter struct {
	rules chessrules.Rules
}

func NewCanonicalJSONAdapter(rules chessrules.Rules) PuzzleAdapter {
	return canonicalJSONAdapter{rules: rules}
}

func (canonicalJSONAdapter) Format() ImportFormat {
	return FormatCanonicalJSON
}

func (a canonicalJSONAdapter) Inspect(
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

	descriptor, err := inspectCanonicalJSONDocument(
		json.NewDecoder(contextReader{ctx: ctx, reader: file}),
	)
	if err != nil {
		return ImportInspection{}, false, fmt.Errorf("inspect canonical JSON: %w", err)
	}
	inspection := ImportInspection{
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

func inspectCanonicalJSONDocument(decoder *json.Decoder) (canonicalSource, error) {
	start, err := decoder.Token()
	if err != nil {
		return canonicalSource{}, err
	}
	if delimiter, ok := start.(json.Delim); !ok || delimiter != '{' {
		return canonicalSource{}, errors.New("canonical JSON document must be an object")
	}

	seen := make(map[string]struct{}, len(canonicalTopLevelFields))
	var source canonicalSource
	for decoder.More() {
		key, err := canonicalObjectKey(decoder, "top-level")
		if err != nil {
			return canonicalSource{}, err
		}
		if err := acceptCanonicalKey(seen, canonicalTopLevelFields, "top-level", key); err != nil {
			return canonicalSource{}, err
		}
		switch key {
		case "schema":
			var schema string
			if err := decoder.Decode(&schema); err != nil {
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
			var raw json.RawMessage
			if err := decoder.Decode(&raw); err != nil {
				return canonicalSource{}, fmt.Errorf("decode source: %w", err)
			}
			source, err = decodeCanonicalSource(raw)
			if err != nil {
				return canonicalSource{}, err
			}
		case "puzzles":
			if err := consumeCanonicalPuzzleArray(decoder); err != nil {
				return canonicalSource{}, err
			}
		}
	}
	if _, err := decoder.Token(); err != nil {
		return canonicalSource{}, fmt.Errorf("close top-level object: %w", err)
	}
	if _, present := seen["schema"]; !present {
		return canonicalSource{}, errors.New("canonical JSON schema is required")
	}
	if _, present := seen["puzzles"]; !present {
		return canonicalSource{}, errors.New("canonical JSON puzzles array is required")
	}
	if err := requireCanonicalEOF(decoder); err != nil {
		return canonicalSource{}, err
	}
	return source, nil
}

func consumeCanonicalPuzzleArray(decoder *json.Decoder) error {
	start, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("open puzzles array: %w", err)
	}
	if delimiter, ok := start.(json.Delim); !ok || delimiter != '[' {
		return errors.New("canonical JSON puzzles must be an array")
	}
	for decoder.More() {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return fmt.Errorf("decode puzzle framing: %w", err)
		}
	}
	end, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("close puzzles array: %w", err)
	}
	if delimiter, ok := end.(json.Delim); !ok || delimiter != ']' {
		return errors.New("canonical JSON puzzles array is not closed")
	}
	return nil
}

func (a canonicalJSONAdapter) NewDecoder(
	reader io.Reader,
	inspection ImportInspection,
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
		decoder:    json.NewDecoder(reader),
		seen:       make(map[string]struct{}, len(canonicalTopLevelFields)),
	}, nil
}

type canonicalJSONDecoder struct {
	rules       chessrules.Rules
	inspection  ImportInspection
	decoder     *json.Decoder
	seen        map[string]struct{}
	initialized bool
	inPuzzles   bool
	finished    bool
	closed      bool
	ordinal     int64
}

func (d *canonicalJSONDecoder) Next(ctx context.Context) (DecodedRecord, error) {
	if err := ctx.Err(); err != nil {
		return DecodedRecord{}, err
	}
	if d.closed || d.finished {
		return DecodedRecord{}, io.EOF
	}
	if !d.initialized {
		start, err := d.decoder.Token()
		if err != nil {
			return DecodedRecord{}, fmt.Errorf("open canonical JSON document: %w", err)
		}
		if delimiter, ok := start.(json.Delim); !ok || delimiter != '{' {
			return DecodedRecord{}, errors.New("canonical JSON document must be an object")
		}
		d.initialized = true
	}

	for {
		if err := ctx.Err(); err != nil {
			return DecodedRecord{}, err
		}
		if d.inPuzzles {
			if d.decoder.More() {
				var raw json.RawMessage
				if err := d.decoder.Decode(&raw); err != nil {
					return DecodedRecord{}, fmt.Errorf(
						"decode canonical JSON puzzle %d framing: %w",
						d.ordinal+1,
						err,
					)
				}
				d.ordinal++
				return d.decodeRecord(raw), nil
			}
			end, err := d.decoder.Token()
			if err != nil {
				return DecodedRecord{}, fmt.Errorf("close puzzles array: %w", err)
			}
			if delimiter, ok := end.(json.Delim); !ok || delimiter != ']' {
				return DecodedRecord{}, errors.New("canonical JSON puzzles array is not closed")
			}
			d.inPuzzles = false
			continue
		}

		if d.decoder.More() {
			key, err := canonicalObjectKey(d.decoder, "top-level")
			if err != nil {
				return DecodedRecord{}, err
			}
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
				start, err := d.decoder.Token()
				if err != nil {
					return DecodedRecord{}, fmt.Errorf("open puzzles array: %w", err)
				}
				if delimiter, ok := start.(json.Delim); !ok || delimiter != '[' {
					return DecodedRecord{}, errors.New("canonical JSON puzzles must be an array")
				}
				d.inPuzzles = true
			}
			continue
		}

		if _, err := d.decoder.Token(); err != nil {
			return DecodedRecord{}, fmt.Errorf("close top-level object: %w", err)
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
		if err := requireCanonicalEOF(d.decoder); err != nil {
			return DecodedRecord{}, err
		}
		d.finished = true
		return DecodedRecord{}, io.EOF
	}
}

func (d *canonicalJSONDecoder) decodeSchema() error {
	var schema string
	if err := d.decoder.Decode(&schema); err != nil {
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
	var raw json.RawMessage
	if err := d.decoder.Decode(&raw); err != nil {
		return fmt.Errorf("decode source: %w", err)
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
	var fields canonicalPuzzle
	seen, err := strictCanonicalObject(raw, "puzzle", canonicalPuzzleFields, &fields)
	if err != nil {
		return TrainingPuzzle{}, err
	}
	if fields.DisplayedFEN == nil {
		return TrainingPuzzle{}, errors.New("displayedFen is required and must be a string")
	}
	if fields.Solver == nil {
		return TrainingPuzzle{}, errors.New("solver is required and must be a string")
	}
	if fields.Solution == nil {
		return TrainingPuzzle{}, errors.New("solution is required and must be an array")
	}
	if err := rejectCanonicalNull(seen, "id", fields.ID != nil); err != nil {
		return TrainingPuzzle{}, err
	}
	if err := rejectCanonicalNull(seen, "sourceFen", fields.SourceFEN != nil); err != nil {
		return TrainingPuzzle{}, err
	}
	if err := rejectCanonicalNull(seen, "preludeUci", fields.PreludeUCI != nil); err != nil {
		return TrainingPuzzle{}, err
	}
	if err := rejectCanonicalNull(seen, "rating", fields.Rating != nil); err != nil {
		return TrainingPuzzle{}, err
	}
	if err := rejectCanonicalNull(seen, "themes", fields.Themes != nil); err != nil {
		return TrainingPuzzle{}, err
	}
	if err := rejectCanonicalNull(seen, "popularity", fields.Popularity != nil); err != nil {
		return TrainingPuzzle{}, err
	}
	if err := rejectCanonicalNull(seen, "playCount", fields.PlayCount != nil); err != nil {
		return TrainingPuzzle{}, err
	}
	if err := rejectCanonicalNull(seen, "url", fields.URL != nil); err != nil {
		return TrainingPuzzle{}, err
	}
	if err := rejectCanonicalNull(seen, "attribution", fields.Attribution != nil); err != nil {
		return TrainingPuzzle{}, err
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

	nodes, err := decodeCanonicalMoves(*fields.Solution, 1, new(int))
	if err != nil {
		return TrainingPuzzle{}, err
	}
	core, err := finalizeCore(d.rules, *fields.DisplayedFEN, solver, nodes)
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
			SourceID:    d.inspection.SourceID,
			SourceKind:  string(FormatCanonicalJSON),
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

type canonicalSource struct {
	ID          string
	Name        string
	URL         string
	Attribution string
}

type canonicalSourceJSON struct {
	ID          *string `json:"id"`
	Name        *string `json:"name"`
	URL         *string `json:"url"`
	Attribution *string `json:"attribution"`
}

func decodeCanonicalSource(raw json.RawMessage) (canonicalSource, error) {
	var fields canonicalSourceJSON
	seen, err := strictCanonicalObject(raw, "source", canonicalSourceFields, &fields)
	if err != nil {
		return canonicalSource{}, err
	}
	for key, value := range map[string]*string{
		"id": fields.ID, "name": fields.Name, "url": fields.URL, "attribution": fields.Attribution,
	} {
		if err := rejectCanonicalNull(seen, key, value != nil); err != nil {
			return canonicalSource{}, err
		}
	}
	value := func(pointer *string) string {
		if pointer == nil {
			return ""
		}
		return strings.TrimSpace(*pointer)
	}
	return canonicalSource{
		ID:          value(fields.ID),
		Name:        value(fields.Name),
		URL:         value(fields.URL),
		Attribution: value(fields.Attribution),
	}, nil
}

type canonicalPuzzle struct {
	ID           *string            `json:"id"`
	SourceFEN    *string            `json:"sourceFen"`
	PreludeUCI   *string            `json:"preludeUci"`
	DisplayedFEN *string            `json:"displayedFen"`
	Solver       *string            `json:"solver"`
	Solution     *[]json.RawMessage `json:"solution"`
	Rating       *int               `json:"rating"`
	Themes       *[]string          `json:"themes"`
	Popularity   *int               `json:"popularity"`
	PlayCount    *int               `json:"playCount"`
	URL          *string            `json:"url"`
	Attribution  *string            `json:"attribution"`
	Metadata     json.RawMessage    `json:"metadata"`
}

type canonicalMove struct {
	UCI      *string            `json:"uci"`
	Children *[]json.RawMessage `json:"children"`
}

func decodeCanonicalMoves(
	rawMoves []json.RawMessage,
	depth int,
	total *int,
) ([]domain.MoveNode, error) {
	if depth > maxSolutionDepth {
		return nil, fmt.Errorf("solution depth exceeds maximum of %d", maxSolutionDepth)
	}
	nodes := make([]domain.MoveNode, 0, len(rawMoves))
	for _, raw := range rawMoves {
		*total++
		if *total > maxSolutionNodes {
			return nil, fmt.Errorf("solution exceeds maximum of %d nodes", maxSolutionNodes)
		}
		var fields canonicalMove
		seen, err := strictCanonicalObject(raw, "move", canonicalMoveFields, &fields)
		if err != nil {
			return nil, err
		}
		if fields.UCI == nil {
			return nil, errors.New("move uci is required and must be a string")
		}
		if err := rejectCanonicalNull(seen, "children", fields.Children != nil); err != nil {
			return nil, err
		}
		node := domain.MoveNode{UCI: strings.ToLower(strings.TrimSpace(*fields.UCI))}
		if fields.Children != nil && len(*fields.Children) > 0 {
			node.Children, err = decodeCanonicalMoves(*fields.Children, depth+1, total)
			if err != nil {
				return nil, err
			}
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func decodeCanonicalMetadata(
	raw json.RawMessage,
	seen map[string]struct{},
) (map[string]any, error) {
	if _, present := seen["metadata"]; !present {
		return nil, nil
	}
	if len(raw) > maxCanonicalJSONMetadataBytes {
		return nil, fmt.Errorf(
			"metadata exceeds maximum of %d bytes",
			maxCanonicalJSONMetadataBytes,
		)
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, errors.New("metadata must be a non-null JSON object")
	}
	var metadata map[string]any
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	if err := decoder.Decode(&metadata); err != nil {
		return nil, fmt.Errorf("decode metadata: %w", err)
	}
	if err := requireCanonicalEOF(decoder); err != nil {
		return nil, fmt.Errorf("decode metadata: %w", err)
	}
	return metadata, nil
}

func strictCanonicalObject(
	raw json.RawMessage,
	label string,
	allowed map[string]struct{},
	target any,
) (map[string]struct{}, error) {
	seen, err := validateCanonicalObjectKeys(raw, label, allowed)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return nil, fmt.Errorf("decode %s: %w", label, err)
	}
	if err := requireCanonicalEOF(decoder); err != nil {
		return nil, fmt.Errorf("decode %s: %w", label, err)
	}
	return seen, nil
}

func validateCanonicalObjectKeys(
	raw json.RawMessage,
	label string,
	allowed map[string]struct{},
) (map[string]struct{}, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	start, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", label, err)
	}
	if delimiter, ok := start.(json.Delim); !ok || delimiter != '{' {
		return nil, fmt.Errorf("%s must be a JSON object", label)
	}
	seen := make(map[string]struct{}, len(allowed))
	for decoder.More() {
		key, err := canonicalObjectKey(decoder, label)
		if err != nil {
			return nil, err
		}
		if err := acceptCanonicalKey(seen, allowed, label, key); err != nil {
			return nil, err
		}
		var discarded json.RawMessage
		if err := decoder.Decode(&discarded); err != nil {
			return nil, fmt.Errorf("decode %s field %q: %w", label, key, err)
		}
	}
	if _, err := decoder.Token(); err != nil {
		return nil, fmt.Errorf("close %s object: %w", label, err)
	}
	if err := requireCanonicalEOF(decoder); err != nil {
		return nil, fmt.Errorf("decode %s: %w", label, err)
	}
	return seen, nil
}

func canonicalObjectKey(decoder *json.Decoder, label string) (string, error) {
	token, err := decoder.Token()
	if err != nil {
		return "", fmt.Errorf("decode %s object key: %w", label, err)
	}
	key, ok := token.(string)
	if !ok {
		return "", fmt.Errorf("decode %s object key: got %T", label, token)
	}
	return key, nil
}

func acceptCanonicalKey(
	seen map[string]struct{},
	allowed map[string]struct{},
	label string,
	key string,
) error {
	if _, duplicate := seen[key]; duplicate {
		return fmt.Errorf("duplicate %s field %q", label, key)
	}
	if _, accepted := allowed[key]; !accepted {
		return fmt.Errorf("unknown %s field %q", label, key)
	}
	seen[key] = struct{}{}
	return nil
}

func rejectCanonicalNull(seen map[string]struct{}, key string, nonNil bool) error {
	if _, present := seen[key]; present && !nonNil {
		return fmt.Errorf("%s must not be null", key)
	}
	return nil
}

func requireCanonicalEOF(decoder *json.Decoder) error {
	if token, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err != nil {
			return err
		}
		return fmt.Errorf("trailing JSON value %v", token)
	}
	return nil
}

func canonicalJSONRejection(ordinal int64, err error) DecodedRecord {
	rejection := Rejection{Ordinal: ordinal, Reason: err.Error()}
	return DecodedRecord{Rejection: &rejection}
}

var _ PuzzleAdapter = canonicalJSONAdapter{}
var _ PuzzleDecoder = (*canonicalJSONDecoder)(nil)
