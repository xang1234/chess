package puzzles

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"chess-trainer/internal/domain"
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

type canonicalSource struct {
	ID          string
	Name        string
	URL         string
	Attribution string
}

func decodeCanonicalSource(raw json.RawMessage) (canonicalSource, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	start, err := decoder.Token()
	if err != nil {
		return canonicalSource{}, fmt.Errorf("decode source: %w", err)
	}
	if delimiter, ok := start.(json.Delim); !ok || delimiter != '{' {
		return canonicalSource{}, errors.New("source must be a JSON object")
	}

	seen := make(map[string]struct{}, len(canonicalSourceFields))
	fields := make(map[string]string, len(canonicalSourceFields))
	for decoder.More() {
		key, err := canonicalObjectKey(decoder, "source")
		if err != nil {
			return canonicalSource{}, err
		}
		if err := acceptCanonicalKey(seen, canonicalSourceFields, "source", key); err != nil {
			return canonicalSource{}, err
		}
		value, err := decodeCanonicalString(decoder, key)
		if err != nil {
			return canonicalSource{}, err
		}
		fields[key] = strings.TrimSpace(*value)
	}
	end, err := decoder.Token()
	if err != nil {
		return canonicalSource{}, fmt.Errorf("close source object: %w", err)
	}
	if delimiter, ok := end.(json.Delim); !ok || delimiter != '}' {
		return canonicalSource{}, errors.New("source object is not closed")
	}
	if err := requireCanonicalEOF(decoder); err != nil {
		return canonicalSource{}, fmt.Errorf("decode source: %w", err)
	}
	return canonicalSource{
		ID:          fields["id"],
		Name:        fields["name"],
		URL:         fields["url"],
		Attribution: fields["attribution"],
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
		*total = *total + 1
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
