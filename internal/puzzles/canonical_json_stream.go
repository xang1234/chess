package puzzles

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const maxCanonicalJSONStructuralKeyBytes = 256

type canonicalJSONObjectState uint8

const (
	canonicalJSONObjectUnopened canonicalJSONObjectState = iota
	canonicalJSONObjectFirstField
	canonicalJSONObjectNextField
	canonicalJSONObjectDone
)

type canonicalJSONArrayState uint8

const (
	canonicalJSONArrayClosed canonicalJSONArrayState = iota
	canonicalJSONArrayFirstItem
	canonicalJSONArrayNextItem
	canonicalJSONArrayDone
)

type canonicalJSONDocumentStream struct {
	reader      *bufio.Reader
	objectState canonicalJSONObjectState
	arrayState  canonicalJSONArrayState
}

func newCanonicalJSONDocumentStream(reader io.Reader) *canonicalJSONDocumentStream {
	return &canonicalJSONDocumentStream{reader: bufio.NewReader(reader)}
}

func (s *canonicalJSONDocumentStream) nextField() (string, bool, error) {
	if s.arrayState != canonicalJSONArrayClosed {
		return "", false, errors.New("cannot read a top-level field while puzzles array is open")
	}
	if s.objectState == canonicalJSONObjectDone {
		return "", true, nil
	}
	if s.objectState == canonicalJSONObjectUnopened {
		start, err := readBoundedJSONNonSpace(s.reader)
		if err != nil {
			return "", false, fmt.Errorf("open canonical JSON document: %w", err)
		}
		if start != '{' {
			return "", false, errors.New("canonical JSON document must be an object")
		}
		s.objectState = canonicalJSONObjectFirstField
	}

	first, err := readBoundedJSONNonSpace(s.reader)
	if err != nil {
		return "", false, fmt.Errorf("read top-level field: %w", err)
	}
	switch s.objectState {
	case canonicalJSONObjectFirstField:
		if first == '}' {
			s.objectState = canonicalJSONObjectDone
			return "", true, nil
		}
		s.objectState = canonicalJSONObjectNextField
	case canonicalJSONObjectNextField:
		if first == '}' {
			s.objectState = canonicalJSONObjectDone
			return "", true, nil
		}
		if first != ',' {
			return "", false, fmt.Errorf(
				"top-level field must be preceded by comma or closing brace, got %q",
				first,
			)
		}
		first, err = readBoundedJSONNonSpace(s.reader)
		if err != nil {
			return "", false, fmt.Errorf("read top-level field after comma: %w", err)
		}
		if first == '}' {
			return "", false, errors.New("top-level object has a trailing comma")
		}
	default:
		return "", false, errors.New("canonical JSON top-level object state is invalid")
	}
	if first != '"' {
		return "", false, fmt.Errorf("top-level object key must be a string, got %q", first)
	}
	capture := newBoundedJSONCapture(maxCanonicalJSONStructuralKeyBytes)
	capture.append(first)
	if err := consumeBoundedJSONString(s.reader, capture); err != nil {
		return "", false, fmt.Errorf("decode top-level object key: %w", err)
	}
	if capture.oversized {
		return "", false, errors.New("top-level object key is too large")
	}
	var key string
	if err := json.Unmarshal(capture.bytes, &key); err != nil {
		return "", false, fmt.Errorf("decode top-level object key: %w", err)
	}
	colon, err := readBoundedJSONNonSpace(s.reader)
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
	if s.arrayState != canonicalJSONArrayClosed {
		return nil, false, errors.New("cannot read a field value while puzzles array is open")
	}
	return readBoundedJSONValue(s.reader, limit)
}

func (s *canonicalJSONDocumentStream) beginArray() error {
	if s.arrayState != canonicalJSONArrayClosed {
		return errors.New("canonical JSON puzzles array is already open")
	}
	start, err := readBoundedJSONNonSpace(s.reader)
	if err != nil {
		return fmt.Errorf("open puzzles array: %w", err)
	}
	if start != '[' {
		return errors.New("canonical JSON puzzles must be an array")
	}
	s.arrayState = canonicalJSONArrayFirstItem
	return nil
}

func (s *canonicalJSONDocumentStream) nextArrayValue(
	limit int,
) ([]byte, bool, bool, error) {
	if s.arrayState == canonicalJSONArrayDone {
		s.arrayState = canonicalJSONArrayClosed
		return nil, false, true, nil
	}
	if s.arrayState == canonicalJSONArrayClosed {
		return nil, false, false, errors.New("canonical JSON puzzles array is not open")
	}

	first, err := readBoundedJSONNonSpace(s.reader)
	if err != nil {
		return nil, false, false, fmt.Errorf("read puzzles array value: %w", err)
	}
	switch s.arrayState {
	case canonicalJSONArrayFirstItem:
		if first == ']' {
			s.arrayState = canonicalJSONArrayClosed
			return nil, false, true, nil
		}
	case canonicalJSONArrayNextItem:
		if first == ']' {
			return nil, false, false, errors.New("puzzles array has a trailing comma")
		}
	default:
		return nil, false, false, errors.New("canonical JSON puzzles array state is invalid")
	}

	capture := newBoundedJSONCapture(limit)
	if err := consumeBoundedJSONValue(s.reader, capture, first, 1); err != nil {
		return nil, false, false, err
	}
	delimiter, err := readBoundedJSONNonSpace(s.reader)
	if err != nil {
		return nil, false, false, fmt.Errorf("read puzzles array delimiter: %w", err)
	}
	switch delimiter {
	case ',':
		s.arrayState = canonicalJSONArrayNextItem
	case ']':
		s.arrayState = canonicalJSONArrayDone
	default:
		return nil, false, false, fmt.Errorf(
			"puzzles array value must be followed by comma or closing bracket, got %q",
			delimiter,
		)
	}
	return capture.bytes, capture.oversized, false, nil
}

func (s *canonicalJSONDocumentStream) requireEOF() error {
	if s.objectState != canonicalJSONObjectDone {
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
		if !isBoundedJSONSpace(value) {
			return fmt.Errorf("trailing JSON value starts with %q", value)
		}
	}
}
