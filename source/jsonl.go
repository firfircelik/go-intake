package source

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/firfircelik/go-intake"
)

// JSONLSource is a Source that reads newline-delimited JSON. Each
// non-empty line is decoded as a JSON object and emitted as a Record.
type JSONLSource struct {
	path   string
	reader io.Reader
	closer io.Closer

	file   *os.File
	sc     *bufio.Scanner
	dec    *json.Decoder
	opened bool
	lineNo int
}

// JSONL returns a JSONLSource that reads from the file at path.
func JSONL(path string) *JSONLSource {
	return &JSONLSource{path: path}
}

// NewJSONLSource returns a JSONLSource that reads from r. If r also
// implements io.Closer, Close will call r.Close.
func NewJSONLSource(r io.Reader) *JSONLSource {
	c, _ := r.(io.Closer)
	return &JSONLSource{reader: r, closer: c}
}

// Open opens the file or reader and prepares the scanner.
func (s *JSONLSource) Open(ctx context.Context) error {
	if s.opened {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if s.reader != nil {
		s.sc = bufio.NewScanner(s.reader)
	} else {
		f, err := os.Open(s.path)
		if err != nil {
			return fmt.Errorf("open jsonl: %w", err)
		}
		s.file = f
		s.sc = bufio.NewScanner(f)
	}
	s.sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	s.opened = true
	return nil
}

// Read returns the next JSON object as a Record.
func (s *JSONLSource) Read(ctx context.Context) (intake.Record, error) {
	if !s.opened {
		return nil, intake.NewError(intake.ErrKindConfig, "source_not_open", "JSONLSource.Read called before Open")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	for s.sc.Scan() {
		s.lineNo++
		line := strings.TrimSpace(s.sc.Text())
		if line == "" {
			continue
		}
		var rec intake.Record
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return nil, fmt.Errorf("parse jsonl line %d: %w", s.lineNo, err)
		}
		if rec == nil {
			continue
		}
		return rec, nil
	}
	if err := s.sc.Err(); err != nil {
		return nil, fmt.Errorf("read jsonl: %w", err)
	}
	return nil, io.EOF
}

// Close releases the underlying file or reader.
func (s *JSONLSource) Close() error {
	var firstErr error
	if s.file != nil {
		if err := s.file.Close(); err != nil {
			firstErr = err
		}
		s.file = nil
	}
	if s.closer != nil {
		if err := s.closer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		s.closer = nil
	}
	s.sc = nil
	s.opened = false
	return firstErr
}
