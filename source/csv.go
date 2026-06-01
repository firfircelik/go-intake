package source

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/firfircelik/go-intake"
)

// CSVSource is a Source that reads RFC 4180-style CSV data. The first
// non-empty row is treated as a header and used as record keys.
//
// Build one with CSV or NewCSVSource.
type CSVSource struct {
	path      string
	reader    io.Reader
	closer    io.Closer
	comma     rune
	lazyOpen  bool
	skipEmpty bool

	file    *os.File
	csv     *csv.Reader
	headers []string
	opened  bool
	lineNo  int
}

// CSV returns a CSVSource that reads from the file at path.
//
// The default field delimiter is ','. Use the option setters on the
// returned value to customise behaviour before passing it to a
// Pipeline.
func CSV(path string) *CSVSource {
	return &CSVSource{path: path, comma: ',', skipEmpty: true}
}

// NewCSVSource returns a CSVSource that reads from r. If r also
// implements io.Closer, Close will call r.Close.
func NewCSVSource(r io.Reader) *CSVSource {
	c, _ := r.(io.Closer)
	return &CSVSource{reader: r, closer: c, comma: ',', skipEmpty: true}
}

// Comma sets the field delimiter. Default is ','.
func (s *CSVSource) Comma(c rune) *CSVSource {
	s.comma = c
	return s
}

// SkipEmptyLines toggles skipping fully-empty rows. Default true.
func (s *CSVSource) SkipEmptyLines(skip bool) *CSVSource {
	s.skipEmpty = skip
	return s
}

// Open opens the underlying file or reader and reads the header row.
func (s *CSVSource) Open(ctx context.Context) error {
	if s.opened {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if s.reader != nil {
		s.csv = csv.NewReader(s.reader)
	} else {
		f, err := os.Open(s.path)
		if err != nil {
			return fmt.Errorf("open csv: %w", err)
		}
		s.file = f
		s.csv = csv.NewReader(f)
	}
	s.csv.Comma = s.comma
	s.csv.FieldsPerRecord = -1
	s.csv.TrimLeadingSpace = true

	header, err := s.csv.Read()
	if err != nil {
		if err == io.EOF {
			s.headers = nil
			s.opened = true
			return nil
		}
		return fmt.Errorf("read csv header: %w", err)
	}
	s.headers = normalizeHeaders(header)
	s.opened = true
	return nil
}

// Read returns the next record. It returns (nil, io.EOF) when the
// stream is exhausted.
func (s *CSVSource) Read(ctx context.Context) (intake.Record, error) {
	if !s.opened {
		return nil, intake.NewError(intake.ErrKindConfig, "source_not_open", "CSVSource.Read called before Open")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.csv == nil {
		return nil, io.EOF
	}
	for {
		row, err := s.csv.Read()
		s.lineNo++
		if err == io.EOF {
			return nil, io.EOF
		}
		if err != nil {
			return nil, fmt.Errorf("read csv row %d: %w", s.lineNo, err)
		}
		if s.skipEmpty && isEmptyRow(row) {
			continue
		}
		rec := make(intake.Record, len(s.headers))
		for i, h := range s.headers {
			if i < len(row) {
				rec[h] = row[i]
			}
		}
		return rec, nil
	}
}

// Close releases the underlying file or reader.
func (s *CSVSource) Close() error {
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
	s.csv = nil
	s.opened = false
	return firstErr
}

// Headers returns the column names as parsed from the first row.
// Available after Open.
func (s *CSVSource) Headers() []string {
	out := make([]string, len(s.headers))
	copy(out, s.headers)
	return out
}

func normalizeHeaders(h []string) []string {
	out := make([]string, len(h))
	for i, v := range h {
		out[i] = strings.TrimSpace(v)
	}
	return out
}

func isEmptyRow(row []string) bool {
	for _, v := range row {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}
