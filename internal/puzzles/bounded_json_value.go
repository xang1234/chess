package puzzles

import (
	"bufio"
	"errors"
	"fmt"
)

const maxBoundedJSONNestingDepth = 10_000
const initialBoundedJSONCaptureBytes = 256

type boundedJSONCapture struct {
	limit     int
	bytes     []byte
	oversized bool
}

func newBoundedJSONCapture(limit int) *boundedJSONCapture {
	if limit < 0 {
		limit = 0
	}
	initialCapacity := min(limit+1, initialBoundedJSONCaptureBytes)
	return &boundedJSONCapture{
		limit: limit,
		bytes: make([]byte, 0, initialCapacity),
	}
}

func (c *boundedJSONCapture) append(value byte) {
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

func readBoundedJSONValue(
	reader *bufio.Reader,
	limit int,
) ([]byte, bool, error) {
	first, err := readBoundedJSONNonSpace(reader)
	if err != nil {
		return nil, false, err
	}
	capture := newBoundedJSONCapture(limit)
	if err := consumeBoundedJSONValue(reader, capture, first, 1); err != nil {
		return nil, false, err
	}
	return capture.bytes, capture.oversized, nil
}

func consumeBoundedJSONValue(
	reader *bufio.Reader,
	capture *boundedJSONCapture,
	first byte,
	depth int,
) error {
	capture.append(first)
	switch first {
	case '{':
		if depth > maxBoundedJSONNestingDepth {
			return fmt.Errorf("JSON nesting exceeds maximum of %d", maxBoundedJSONNestingDepth)
		}
		return consumeBoundedJSONObject(reader, capture, depth)
	case '[':
		if depth > maxBoundedJSONNestingDepth {
			return fmt.Errorf("JSON nesting exceeds maximum of %d", maxBoundedJSONNestingDepth)
		}
		return consumeBoundedJSONArray(reader, capture, depth)
	case '"':
		return consumeBoundedJSONString(reader, capture)
	case 't':
		return consumeBoundedJSONLiteral(reader, capture, "rue")
	case 'f':
		return consumeBoundedJSONLiteral(reader, capture, "alse")
	case 'n':
		return consumeBoundedJSONLiteral(reader, capture, "ull")
	case '-':
		return consumeBoundedJSONNumber(reader, capture, first)
	default:
		if first >= '0' && first <= '9' {
			return consumeBoundedJSONNumber(reader, capture, first)
		}
		return fmt.Errorf("invalid JSON value start %q", first)
	}
}

func consumeBoundedJSONObject(
	reader *bufio.Reader,
	capture *boundedJSONCapture,
	depth int,
) error {
	next, err := readBoundedJSONNonSpaceCaptured(reader, capture)
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
		if err := consumeBoundedJSONString(reader, capture); err != nil {
			return err
		}
		colon, err := readBoundedJSONNonSpaceCaptured(reader, capture)
		if err != nil {
			return fmt.Errorf("read JSON object colon: %w", err)
		}
		if colon != ':' {
			return fmt.Errorf("JSON object key must be followed by colon, got %q", colon)
		}
		valueStart, err := readBoundedJSONValueStart(reader, capture)
		if err != nil {
			return fmt.Errorf("read JSON object value: %w", err)
		}
		if err := consumeBoundedJSONValue(reader, capture, valueStart, depth+1); err != nil {
			return err
		}
		next, err = readBoundedJSONNonSpaceCaptured(reader, capture)
		if err != nil {
			return fmt.Errorf("read JSON object delimiter: %w", err)
		}
		switch next {
		case '}':
			return nil
		case ',':
			next, err = readBoundedJSONNonSpaceCaptured(reader, capture)
			if err != nil {
				return fmt.Errorf("read JSON object key: %w", err)
			}
		default:
			return fmt.Errorf("JSON object value must be followed by comma or closing brace, got %q", next)
		}
	}
}

func consumeBoundedJSONArray(
	reader *bufio.Reader,
	capture *boundedJSONCapture,
	depth int,
) error {
	next, err := readBoundedJSONValueStart(reader, capture)
	if err != nil {
		return fmt.Errorf("read JSON array: %w", err)
	}
	if next == ']' {
		capture.append(next)
		return nil
	}
	for {
		if err := consumeBoundedJSONValue(reader, capture, next, depth+1); err != nil {
			return err
		}
		next, err = readBoundedJSONNonSpaceCaptured(reader, capture)
		if err != nil {
			return fmt.Errorf("read JSON array delimiter: %w", err)
		}
		switch next {
		case ']':
			return nil
		case ',':
			next, err = readBoundedJSONValueStart(reader, capture)
			if err != nil {
				return fmt.Errorf("read JSON array value: %w", err)
			}
		default:
			return fmt.Errorf("JSON array value must be followed by comma or closing bracket, got %q", next)
		}
	}
}

func consumeBoundedJSONString(
	reader *bufio.Reader,
	capture *boundedJSONCapture,
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
					if !isBoundedJSONHexDigit(hexDigit) {
						return fmt.Errorf("invalid JSON unicode escape digit %q", hexDigit)
					}
				}
			default:
				return fmt.Errorf("invalid JSON string escape %q", escape)
			}
		}
	}
}

func consumeBoundedJSONLiteral(
	reader *bufio.Reader,
	capture *boundedJSONCapture,
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

func consumeBoundedJSONNumber(
	reader *bufio.Reader,
	capture *boundedJSONCapture,
	first byte,
) error {
	integerStart := first
	if first == '-' {
		var err error
		integerStart, err = readBoundedJSONNumberByte(reader, capture)
		if err != nil {
			return err
		}
	}
	if integerStart == '0' {
		if next, err := peekBoundedJSONByte(reader); err == nil && next >= '0' && next <= '9' {
			return errors.New("JSON number has a leading zero")
		}
	} else if integerStart >= '1' && integerStart <= '9' {
		consumeBoundedJSONDigits(reader, capture)
	} else {
		return fmt.Errorf("invalid JSON number digit %q", integerStart)
	}

	if next, err := peekBoundedJSONByte(reader); err == nil && next == '.' {
		_, _ = readBoundedJSONNumberByte(reader, capture)
		fractionStart, err := readBoundedJSONNumberByte(reader, capture)
		if err != nil || fractionStart < '0' || fractionStart > '9' {
			return errors.New("JSON number fraction requires a digit")
		}
		consumeBoundedJSONDigits(reader, capture)
	}
	if next, err := peekBoundedJSONByte(reader); err == nil && (next == 'e' || next == 'E') {
		_, _ = readBoundedJSONNumberByte(reader, capture)
		if sign, err := peekBoundedJSONByte(reader); err == nil && (sign == '+' || sign == '-') {
			_, _ = readBoundedJSONNumberByte(reader, capture)
		}
		exponentStart, err := readBoundedJSONNumberByte(reader, capture)
		if err != nil || exponentStart < '0' || exponentStart > '9' {
			return errors.New("JSON number exponent requires a digit")
		}
		consumeBoundedJSONDigits(reader, capture)
	}
	return nil
}

func consumeBoundedJSONDigits(reader *bufio.Reader, capture *boundedJSONCapture) {
	for {
		next, err := peekBoundedJSONByte(reader)
		if err != nil || next < '0' || next > '9' {
			return
		}
		_, _ = readBoundedJSONNumberByte(reader, capture)
	}
}

func readBoundedJSONNumberByte(
	reader *bufio.Reader,
	capture *boundedJSONCapture,
) (byte, error) {
	value, err := reader.ReadByte()
	if err != nil {
		return 0, fmt.Errorf("read JSON number: %w", err)
	}
	capture.append(value)
	return value, nil
}

func peekBoundedJSONByte(reader *bufio.Reader) (byte, error) {
	buffer, err := reader.Peek(1)
	if err != nil {
		return 0, err
	}
	return buffer[0], nil
}

func readBoundedJSONNonSpace(reader *bufio.Reader) (byte, error) {
	for {
		value, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}
		if !isBoundedJSONSpace(value) {
			return value, nil
		}
	}
}

func readBoundedJSONNonSpaceCaptured(
	reader *bufio.Reader,
	capture *boundedJSONCapture,
) (byte, error) {
	for {
		value, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}
		capture.append(value)
		if !isBoundedJSONSpace(value) {
			return value, nil
		}
	}
}

func readBoundedJSONValueStart(
	reader *bufio.Reader,
	capture *boundedJSONCapture,
) (byte, error) {
	for {
		value, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}
		if !isBoundedJSONSpace(value) {
			return value, nil
		}
		capture.append(value)
	}
}

func isBoundedJSONSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\r' || value == '\n'
}

func isBoundedJSONHexDigit(value byte) bool {
	return value >= '0' && value <= '9' ||
		value >= 'a' && value <= 'f' ||
		value >= 'A' && value <= 'F'
}
