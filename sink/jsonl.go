package sink

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/firfircelik/go-intake"
)

// JSONLSink is a Sink that writes records as newline-delimited JSON.
// Each record is encoded on a single line with a trailing newline.
type JSONLSink struct {
	path string

	file *os.File
	enc  *json.Encoder
}

// JSONL returns a JSONLSink that writes to the file at path. The file
// is truncated on open.
func JSONL(path string) *JSONLSink {
	return &JSONLSink{path: path}
}

// Open opens the destination file for writing.
func (s *JSONLSink) Open(ctx context.Context) error {
	if s.file != nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	f, err := os.Create(s.path)
	if err != nil {
		return fmt.Errorf("create jsonl: %w", err)
	}
	s.file = f
	s.enc = json.NewEncoder(f)
	s.enc.SetEscapeHTML(false)
	return nil
}

// Write encodes a single record as one JSON object followed by a
// newline.
func (s *JSONLSink) Write(ctx context.Context, r intake.Record) error {
	if s.enc == nil {
		return intake.NewError(intake.ErrKindConfig, "sink_not_open", "JSONLSink.Write called before Open")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.enc.Encode(r); err != nil {
		return fmt.Errorf("encode jsonl: %w", err)
	}
	return nil
}

// Close flushes and closes the destination file.
func (s *JSONLSink) Close() error {
	var firstErr error
	if s.file != nil {
		if err := s.file.Sync(); err != nil {
			firstErr = err
		}
		if err := s.file.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		s.file = nil
	}
	s.enc = nil
	return firstErr
}
