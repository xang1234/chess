package openings

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

func DecodeCoursePack(reader io.Reader) (CoursePack, error) {
	if reader == nil {
		return CoursePack{}, errors.New("course pack reader is required")
	}
	var raw bytes.Buffer
	decoder := json.NewDecoder(io.TeeReader(reader, &raw))
	decoder.DisallowUnknownFields()

	var pack CoursePack
	if err := decoder.Decode(&pack); err != nil {
		return CoursePack{}, fmt.Errorf("decode course pack: %w", err)
	}
	if err := requireCoursePackEOF(decoder); err != nil {
		return CoursePack{}, err
	}
	if !utf8.Valid(raw.Bytes()) {
		return CoursePack{}, errors.New("course pack is not valid UTF-8")
	}
	if pack.SchemaVersion != 1 && pack.SchemaVersion != 2 {
		return CoursePack{}, fmt.Errorf("unsupported course schema version %d", pack.SchemaVersion)
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "courseId", value: pack.CourseID},
		{name: "contentVersion", value: pack.ContentVersion},
		{name: "title", value: pack.Title},
		{name: "description", value: pack.Description},
		{name: "rootPositionId", value: pack.RootPositionID},
		{name: "rootFen", value: pack.RootFEN},
		{name: "source.title", value: pack.Source.Title},
		{name: "source.edition", value: pack.Source.Edition},
		{name: "source.privateUseNotice", value: pack.Source.PrivateUseNotice},
	} {
		if strings.TrimSpace(required.value) == "" {
			return CoursePack{}, fmt.Errorf("%s is required", required.name)
		}
	}
	return pack, nil
}

func requireCoursePackEOF(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	switch {
	case errors.Is(err, io.EOF):
		return nil
	case err == nil:
		return errors.New("course pack contains a second JSON value")
	default:
		return fmt.Errorf("decode trailing course pack data: %w", err)
	}
}
