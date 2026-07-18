package puzzles

import (
	"bufio"
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
		contextReader{ctx: ctx, reader: file},
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

const maxCanonicalJSONStructuralKeyBytes = 256

type canonicalJSONDocumentStream struct {
	reader        *bufio.Reader
	objectStarted bool
	firstField    bool
	objectDone    bool
	arrayOpen     bool
	firstItem     bool
	itemReady     bool
	arrayDone     bool
}

func newCanonicalJSONDocumentStream(reader io.Reader) *canonicalJSONDocumentStream {
	return &canonicalJSONDocumentStream{reader: bufio.NewReader(reader)}
}

func (s *canonicalJSONDocumentStream) nextField() (string, bool, error) {
	if s.arrayOpen {
		return "", false, errors.New("cannot read a top-level field while puzzles array is open")
	}
	if s.objectDone {
		return "", true, nil
	}
	if !s.objectStarted {
		start, err := readCanonicalJSONNonSpace(s.reader)
		if err != nil {
			return "", false, fmt.Errorf("open canonical JSON document: %w", err)
		}
		if start != '{' {
			return "", false, errors.New("canonical JSON document must be an object")
		}
		s.objectStarted = true
		s.firstField = true
	}

	first, err := readCanonicalJSONNonSpace(s.reader)
	if err != nil {
		return "", false, fmt.Errorf("read top-level field: %w", err)
	}
	if s.firstField {
		s.firstField = false
		if first == '}' {
			s.objectDone = true
			return "", true, nil
		}
	} else {
		if first == '}' {
			s.objectDone = true
			return "", true, nil
		}
		if first != ',' {
			return "", false, fmt.Errorf(
				"top-level field must be preceded by comma or closing brace, got %q",
				first,
			)
		}
		first, err = readCanonicalJSONNonSpace(s.reader)
		if err != nil {
			return "", false, fmt.Errorf("read top-level field after comma: %w", err)
		}
		if first == '}' {
			return "", false, errors.New("top-level object has a trailing comma")
		}
	}
	if first != '"' {
		return "", false, fmt.Errorf("top-level object key must be a string, got %q", first)
	}
	capture := newCanonicalJSONCapture(maxCanonicalJSONStructuralKeyBytes)
	capture.append(first)
	if err := consumeCanonicalJSONString(s.reader, capture); err != nil {
		return "", false, fmt.Errorf("decode top-level object key: %w", err)
	}
	if capture.oversized {
		return "", false, errors.New("top-level object key is too large")
	}
	var key string
	if err := json.Unmarshal(capture.bytes, &key); err != nil {
		return "", false, fmt.Errorf("decode top-level object key: %w", err)
	}
	colon, err := readCanonicalJSONNonSpace(s.reader)
	if err != nil {
		return "", false, fmt.Errorf("read top-level field %q colon: %w", key, err)
	}
	if colon != ':' {
		return "", false, fmt.Errorf("top-level field %q must be followed by colon", key)
	}
	return key, false, nil
}

func (s *canonicalJSONDocumentStream) readFieldValue(
	limit int,
) ([]byte, bool, error) {
	if s.arrayOpen {
		return nil, false, errors.New("cannot read a field value while puzzles array is open")
	}
	return readCanonicalJSONValue(s.reader, limit)
}

func (s *canonicalJSONDocumentStream) beginArray() error {
	if s.arrayOpen {
		return errors.New("canonical JSON puzzles array is already open")
	}
	start, err := readCanonicalJSONNonSpace(s.reader)
	if err != nil {
		return fmt.Errorf("open puzzles array: %w", err)
	}
	if start != '[' {
		return errors.New("canonical JSON puzzles must be an array")
	}
	s.arrayOpen = true
	s.firstItem = true
	s.itemReady = false
	s.arrayDone = false
	return nil
}

func (s *canonicalJSONDocumentStream) nextArrayValue(
	limit int,
) ([]byte, bool, bool, error) {
	if s.arrayDone {
		s.arrayDone = false
		return nil, false, true, nil
	}
	if !s.arrayOpen {
		return nil, false, false, errors.New("canonical JSON puzzles array is not open")
	}
	var first byte
	var err error
	if s.firstItem {
		s.firstItem = false
		first, err = readCanonicalJSONNonSpace(s.reader)
		if err != nil {
			return nil, false, false, fmt.Errorf("read puzzles array value: %w", err)
		}
		if first == ']' {
			s.arrayOpen = false
			return nil, false, true, nil
		}
	} else if s.itemReady {
		s.itemReady = false
		first, err = readCanonicalJSONNonSpace(s.reader)
		if err != nil {
			return nil, false, false, fmt.Errorf("read puzzles array value after comma: %w", err)
		}
		if first == ']' {
			return nil, false, false, errors.New("puzzles array has a trailing comma")
		}
	} else {
		return nil, false, false, errors.New("canonical JSON puzzles array state is invalid")
	}
	capture := newCanonicalJSONCapture(limit)
	if err := consumeCanonicalJSONValue(s.reader, capture, first, 1); err != nil {
		return nil, false, false, err
	}
	delimiter, err := readCanonicalJSONNonSpace(s.reader)
	if err != nil {
		return nil, false, false, fmt.Errorf("read puzzles array delimiter: %w", err)
	}
	switch delimiter {
	case ',':
		s.itemReady = true
	case ']':
		s.arrayOpen = false
		s.arrayDone = true
	default:
		return nil, false, false, fmt.Errorf(
			"puzzles array value must be followed by comma or closing bracket, got %q",
			delimiter,
		)
	}
	return capture.bytes, capture.oversized, false, nil
}

func (s *canonicalJSONDocumentStream) requireEOF() error {
	if !s.objectDone {
		return errors.New("canonical JSON top-level object is not closed")
	}
	for {
		value, err := s.reader.ReadByte()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if !isCanonicalJSONSpace(value) {
			return fmt.Errorf("trailing JSON value starts with %q", value)
		}
	}
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
		stream:     newCanonicalJSONDocumentStream(reader),
		seen:       make(map[string]struct{}, len(canonicalTopLevelFields)),
	}, nil
}

type canonicalJSONDecoder struct {
	rules      chessrules.Rules
	inspection ImportInspection
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
	ID           *string
	SourceFEN    *string
	PreludeUCI   *string
	DisplayedFEN *string
	Solver       *string
	Solution     []domain.MoveNode
	Rating       *int
	Themes       *[]string
	Popularity   *int
	PlayCount    *int
	URL          *string
	Attribution  *string
	Metadata     json.RawMessage
}

func decodeCanonicalPuzzle(
	raw json.RawMessage,
) (canonicalPuzzle, map[string]struct{}, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	start, err := decoder.Token()
	if err != nil {
		return canonicalPuzzle{}, nil, fmt.Errorf("decode puzzle: %w", err)
	}
	if delimiter, ok := start.(json.Delim); !ok || delimiter != '{' {
		return canonicalPuzzle{}, nil, errors.New("puzzle must be a JSON object")
	}

	fields := canonicalPuzzle{}
	seen := make(map[string]struct{}, len(canonicalPuzzleFields))
	for decoder.More() {
		key, err := canonicalObjectKey(decoder, "puzzle")
		if err != nil {
			return canonicalPuzzle{}, nil, err
		}
		if err := acceptCanonicalKey(seen, canonicalPuzzleFields, "puzzle", key); err != nil {
			return canonicalPuzzle{}, nil, err
		}
		switch key {
		case "id":
			fields.ID, err = decodeCanonicalString(decoder, key)
		case "sourceFen":
			fields.SourceFEN, err = decodeCanonicalString(decoder, key)
		case "preludeUci":
			fields.PreludeUCI, err = decodeCanonicalString(decoder, key)
		case "displayedFen":
			fields.DisplayedFEN, err = decodeCanonicalString(decoder, key)
		case "solver":
			fields.Solver, err = decodeCanonicalString(decoder, key)
		case "solution":
			fields.Solution, err = decodeCanonicalSolution(decoder, 1, new(int))
		case "rating":
			fields.Rating, err = decodeCanonicalInteger(decoder, key)
		case "themes":
			fields.Themes, err = decodeCanonicalStringArray(decoder, key)
		case "popularity":
			fields.Popularity, err = decodeCanonicalInteger(decoder, key)
		case "playCount":
			fields.PlayCount, err = decodeCanonicalInteger(decoder, key)
		case "url":
			fields.URL, err = decodeCanonicalString(decoder, key)
		case "attribution":
			fields.Attribution, err = decodeCanonicalString(decoder, key)
		case "metadata":
			err = decoder.Decode(&fields.Metadata)
			if err != nil {
				err = fmt.Errorf("decode metadata: %w", err)
			}
		}
		if err != nil {
			return canonicalPuzzle{}, nil, err
		}
	}
	end, err := decoder.Token()
	if err != nil {
		return canonicalPuzzle{}, nil, fmt.Errorf("close puzzle object: %w", err)
	}
	if delimiter, ok := end.(json.Delim); !ok || delimiter != '}' {
		return canonicalPuzzle{}, nil, errors.New("puzzle object is not closed")
	}
	if err := requireCanonicalEOF(decoder); err != nil {
		return canonicalPuzzle{}, nil, fmt.Errorf("decode puzzle: %w", err)
	}
	return fields, seen, nil
}

func decodeCanonicalSolution(
	decoder *json.Decoder,
	depth int,
	total *int,
) ([]domain.MoveNode, error) {
	start, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("decode solution: %w", err)
	}
	if delimiter, ok := start.(json.Delim); !ok || delimiter != '[' {
		return nil, errors.New("solution must be an array")
	}
	nodes := make([]domain.MoveNode, 0)
	for decoder.More() {
		if depth > maxSolutionDepth {
			return nil, fmt.Errorf("solution depth exceeds maximum of %d", maxSolutionDepth)
		}
		*total++
		if *total > maxSolutionNodes {
			return nil, fmt.Errorf("solution exceeds maximum of %d nodes", maxSolutionNodes)
		}
		node, err := decodeCanonicalMove(decoder, depth, total)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	end, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("close solution array: %w", err)
	}
	if delimiter, ok := end.(json.Delim); !ok || delimiter != ']' {
		return nil, errors.New("solution array is not closed")
	}
	return nodes, nil
}

func decodeCanonicalMove(
	decoder *json.Decoder,
	depth int,
	total *int,
) (domain.MoveNode, error) {
	start, err := decoder.Token()
	if err != nil {
		return domain.MoveNode{}, fmt.Errorf("decode move: %w", err)
	}
	if delimiter, ok := start.(json.Delim); !ok || delimiter != '{' {
		return domain.MoveNode{}, errors.New("move must be a JSON object")
	}
	seen := make(map[string]struct{}, len(canonicalMoveFields))
	var uci *string
	var children []domain.MoveNode
	for decoder.More() {
		key, err := canonicalObjectKey(decoder, "move")
		if err != nil {
			return domain.MoveNode{}, err
		}
		if err := acceptCanonicalKey(seen, canonicalMoveFields, "move", key); err != nil {
			return domain.MoveNode{}, err
		}
		switch key {
		case "uci":
			uci, err = decodeCanonicalString(decoder, key)
		case "children":
			children, err = decodeCanonicalSolution(decoder, depth+1, total)
		}
		if err != nil {
			return domain.MoveNode{}, err
		}
	}
	end, err := decoder.Token()
	if err != nil {
		return domain.MoveNode{}, fmt.Errorf("close move object: %w", err)
	}
	if delimiter, ok := end.(json.Delim); !ok || delimiter != '}' {
		return domain.MoveNode{}, errors.New("move object is not closed")
	}
	if uci == nil {
		return domain.MoveNode{}, errors.New("move uci is required and must be a string")
	}
	return domain.MoveNode{
		UCI:      strings.ToLower(strings.TrimSpace(*uci)),
		Children: children,
	}, nil
}

func decodeCanonicalString(decoder *json.Decoder, label string) (*string, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", label, err)
	}
	value, ok := token.(string)
	if !ok {
		return nil, fmt.Errorf("%s must be a non-null string", label)
	}
	return &value, nil
}

func decodeCanonicalInteger(decoder *json.Decoder, label string) (*int, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", label, err)
	}
	number, ok := token.(json.Number)
	if !ok {
		return nil, fmt.Errorf("%s must be a non-null integer", label)
	}
	value, err := strconv.Atoi(number.String())
	if err != nil {
		return nil, fmt.Errorf("%s must be an integer: %w", label, err)
	}
	return &value, nil
}

func decodeCanonicalStringArray(
	decoder *json.Decoder,
	label string,
) (*[]string, error) {
	start, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", label, err)
	}
	if delimiter, ok := start.(json.Delim); !ok || delimiter != '[' {
		return nil, fmt.Errorf("%s must be a non-null array", label)
	}
	values := make([]string, 0)
	for decoder.More() {
		value, err := decodeCanonicalString(decoder, label+" item")
		if err != nil {
			return nil, err
		}
		values = append(values, *value)
	}
	end, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("close %s array: %w", label, err)
	}
	if delimiter, ok := end.(json.Delim); !ok || delimiter != ']' {
		return nil, fmt.Errorf("%s array is not closed", label)
	}
	return &values, nil
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

const maxCanonicalJSONNestingDepth = 10_000
const initialCanonicalJSONCaptureBytes = 256

type canonicalJSONCapture struct {
	limit     int
	bytes     []byte
	oversized bool
}

func newCanonicalJSONCapture(limit int) *canonicalJSONCapture {
	if limit < 0 {
		limit = 0
	}
	initialCapacity := min(limit+1, initialCanonicalJSONCaptureBytes)
	return &canonicalJSONCapture{
		limit: limit,
		bytes: make([]byte, 0, initialCapacity),
	}
}

func (c *canonicalJSONCapture) append(value byte) {
	maximum := c.limit + 1
	if len(c.bytes) >= maximum {
		c.oversized = true
		return
	}
	if len(c.bytes) == cap(c.bytes) {
		capacity := max(1, cap(c.bytes)*2)
		capacity = min(capacity, maximum)
		grown := make([]byte, len(c.bytes), capacity)
		copy(grown, c.bytes)
		c.bytes = grown
	}
	c.bytes = append(c.bytes, value)
	if len(c.bytes) > c.limit {
		c.oversized = true
	}
}

func readCanonicalJSONValue(
	reader *bufio.Reader,
	limit int,
) ([]byte, bool, error) {
	first, err := readCanonicalJSONNonSpace(reader)
	if err != nil {
		return nil, false, err
	}
	capture := newCanonicalJSONCapture(limit)
	if err := consumeCanonicalJSONValue(reader, capture, first, 1); err != nil {
		return nil, false, err
	}
	return capture.bytes, capture.oversized, nil
}

func consumeCanonicalJSONValue(
	reader *bufio.Reader,
	capture *canonicalJSONCapture,
	first byte,
	depth int,
) error {
	capture.append(first)
	switch first {
	case '{':
		if depth > maxCanonicalJSONNestingDepth {
			return fmt.Errorf("JSON nesting exceeds maximum of %d", maxCanonicalJSONNestingDepth)
		}
		return consumeCanonicalJSONObject(reader, capture, depth)
	case '[':
		if depth > maxCanonicalJSONNestingDepth {
			return fmt.Errorf("JSON nesting exceeds maximum of %d", maxCanonicalJSONNestingDepth)
		}
		return consumeCanonicalJSONArray(reader, capture, depth)
	case '"':
		return consumeCanonicalJSONString(reader, capture)
	case 't':
		return consumeCanonicalJSONLiteral(reader, capture, "rue")
	case 'f':
		return consumeCanonicalJSONLiteral(reader, capture, "alse")
	case 'n':
		return consumeCanonicalJSONLiteral(reader, capture, "ull")
	case '-':
		return consumeCanonicalJSONNumber(reader, capture, first)
	default:
		if first >= '0' && first <= '9' {
			return consumeCanonicalJSONNumber(reader, capture, first)
		}
		return fmt.Errorf("invalid JSON value start %q", first)
	}
}

func consumeCanonicalJSONObject(
	reader *bufio.Reader,
	capture *canonicalJSONCapture,
	depth int,
) error {
	next, err := readCanonicalJSONNonSpaceCaptured(reader, capture)
	if err != nil {
		return fmt.Errorf("read JSON object: %w", err)
	}
	if next == '}' {
		return nil
	}
	for {
		if next != '"' {
			return fmt.Errorf("JSON object key must be a string, got %q", next)
		}
		if err := consumeCanonicalJSONString(reader, capture); err != nil {
			return err
		}
		colon, err := readCanonicalJSONNonSpaceCaptured(reader, capture)
		if err != nil {
			return fmt.Errorf("read JSON object colon: %w", err)
		}
		if colon != ':' {
			return fmt.Errorf("JSON object key must be followed by colon, got %q", colon)
		}
		valueStart, err := readCanonicalJSONValueStart(reader, capture)
		if err != nil {
			return fmt.Errorf("read JSON object value: %w", err)
		}
		if err := consumeCanonicalJSONValue(reader, capture, valueStart, depth+1); err != nil {
			return err
		}
		next, err = readCanonicalJSONNonSpaceCaptured(reader, capture)
		if err != nil {
			return fmt.Errorf("read JSON object delimiter: %w", err)
		}
		switch next {
		case '}':
			return nil
		case ',':
			next, err = readCanonicalJSONNonSpaceCaptured(reader, capture)
			if err != nil {
				return fmt.Errorf("read JSON object key: %w", err)
			}
		default:
			return fmt.Errorf("JSON object value must be followed by comma or closing brace, got %q", next)
		}
	}
}

func consumeCanonicalJSONArray(
	reader *bufio.Reader,
	capture *canonicalJSONCapture,
	depth int,
) error {
	next, err := readCanonicalJSONValueStart(reader, capture)
	if err != nil {
		return fmt.Errorf("read JSON array: %w", err)
	}
	if next == ']' {
		capture.append(next)
		return nil
	}
	for {
		if err := consumeCanonicalJSONValue(reader, capture, next, depth+1); err != nil {
			return err
		}
		next, err = readCanonicalJSONNonSpaceCaptured(reader, capture)
		if err != nil {
			return fmt.Errorf("read JSON array delimiter: %w", err)
		}
		switch next {
		case ']':
			return nil
		case ',':
			next, err = readCanonicalJSONValueStart(reader, capture)
			if err != nil {
				return fmt.Errorf("read JSON array value: %w", err)
			}
		default:
			return fmt.Errorf("JSON array value must be followed by comma or closing bracket, got %q", next)
		}
	}
}

func consumeCanonicalJSONString(
	reader *bufio.Reader,
	capture *canonicalJSONCapture,
) error {
	for {
		value, err := reader.ReadByte()
		if err != nil {
			return fmt.Errorf("read JSON string: %w", err)
		}
		capture.append(value)
		switch {
		case value == '"':
			return nil
		case value < 0x20:
			return fmt.Errorf("JSON string contains control byte 0x%02x", value)
		case value == '\\':
			escape, err := reader.ReadByte()
			if err != nil {
				return fmt.Errorf("read JSON string escape: %w", err)
			}
			capture.append(escape)
			switch escape {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
			case 'u':
				for index := 0; index < 4; index++ {
					hexDigit, err := reader.ReadByte()
					if err != nil {
						return fmt.Errorf("read JSON unicode escape: %w", err)
					}
					capture.append(hexDigit)
					if !isCanonicalJSONHexDigit(hexDigit) {
						return fmt.Errorf("invalid JSON unicode escape digit %q", hexDigit)
					}
				}
			default:
				return fmt.Errorf("invalid JSON string escape %q", escape)
			}
		}
	}
}

func consumeCanonicalJSONLiteral(
	reader *bufio.Reader,
	capture *canonicalJSONCapture,
	remainder string,
) error {
	for index := range len(remainder) {
		value, err := reader.ReadByte()
		if err != nil {
			return fmt.Errorf("read JSON literal: %w", err)
		}
		capture.append(value)
		if value != remainder[index] {
			return fmt.Errorf("invalid JSON literal byte %q", value)
		}
	}
	return nil
}

func consumeCanonicalJSONNumber(
	reader *bufio.Reader,
	capture *canonicalJSONCapture,
	first byte,
) error {
	integerStart := first
	if first == '-' {
		var err error
		integerStart, err = readCanonicalJSONNumberByte(reader, capture)
		if err != nil {
			return err
		}
	}
	if integerStart == '0' {
		if next, err := peekCanonicalJSONByte(reader); err == nil && next >= '0' && next <= '9' {
			return errors.New("JSON number has a leading zero")
		}
	} else if integerStart >= '1' && integerStart <= '9' {
		consumeCanonicalJSONDigits(reader, capture)
	} else {
		return fmt.Errorf("invalid JSON number digit %q", integerStart)
	}

	if next, err := peekCanonicalJSONByte(reader); err == nil && next == '.' {
		_, _ = readCanonicalJSONNumberByte(reader, capture)
		fractionStart, err := readCanonicalJSONNumberByte(reader, capture)
		if err != nil || fractionStart < '0' || fractionStart > '9' {
			return errors.New("JSON number fraction requires a digit")
		}
		consumeCanonicalJSONDigits(reader, capture)
	}
	if next, err := peekCanonicalJSONByte(reader); err == nil && (next == 'e' || next == 'E') {
		_, _ = readCanonicalJSONNumberByte(reader, capture)
		if sign, err := peekCanonicalJSONByte(reader); err == nil && (sign == '+' || sign == '-') {
			_, _ = readCanonicalJSONNumberByte(reader, capture)
		}
		exponentStart, err := readCanonicalJSONNumberByte(reader, capture)
		if err != nil || exponentStart < '0' || exponentStart > '9' {
			return errors.New("JSON number exponent requires a digit")
		}
		consumeCanonicalJSONDigits(reader, capture)
	}
	return nil
}

func consumeCanonicalJSONDigits(reader *bufio.Reader, capture *canonicalJSONCapture) {
	for {
		next, err := peekCanonicalJSONByte(reader)
		if err != nil || next < '0' || next > '9' {
			return
		}
		_, _ = readCanonicalJSONNumberByte(reader, capture)
	}
}

func readCanonicalJSONNumberByte(
	reader *bufio.Reader,
	capture *canonicalJSONCapture,
) (byte, error) {
	value, err := reader.ReadByte()
	if err != nil {
		return 0, fmt.Errorf("read JSON number: %w", err)
	}
	capture.append(value)
	return value, nil
}

func peekCanonicalJSONByte(reader *bufio.Reader) (byte, error) {
	buffer, err := reader.Peek(1)
	if err != nil {
		return 0, err
	}
	return buffer[0], nil
}

func readCanonicalJSONNonSpace(reader *bufio.Reader) (byte, error) {
	for {
		value, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}
		if !isCanonicalJSONSpace(value) {
			return value, nil
		}
	}
}

func readCanonicalJSONNonSpaceCaptured(
	reader *bufio.Reader,
	capture *canonicalJSONCapture,
) (byte, error) {
	for {
		value, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}
		capture.append(value)
		if !isCanonicalJSONSpace(value) {
			return value, nil
		}
	}
}

func readCanonicalJSONValueStart(
	reader *bufio.Reader,
	capture *canonicalJSONCapture,
) (byte, error) {
	for {
		value, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}
		if !isCanonicalJSONSpace(value) {
			return value, nil
		}
		capture.append(value)
	}
}

func isCanonicalJSONSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\r' || value == '\n'
}

func isCanonicalJSONHexDigit(value byte) bool {
	return value >= '0' && value <= '9' ||
		value >= 'a' && value <= 'f' ||
		value >= 'A' && value <= 'F'
}

var _ PuzzleAdapter = canonicalJSONAdapter{}
var _ PuzzleDecoder = (*canonicalJSONDecoder)(nil)
