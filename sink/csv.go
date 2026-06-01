package sink

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/firfircelik/go-intake"
)

// CSVSink is a Sink that writes records to a CSV file. The column
// order is determined by the first record written, or by an explicit
// header list passed to WithHeaders.
type CSVSink struct {
	path    string
	comma   rune
	headers []string

	file   *os.File
	writer *csv.Writer
	wrote  bool
}

// CSV returns a CSVSink that writes to the file at path. The file is
// truncated on open.
func CSV(path string) *CSVSink {
	return &CSVSink{path: path, comma: ','}
}

// WithHeaders sets an explicit header row. The records are written in
// this column order and any record missing a column emits an empty
// field. If unset, the headers are taken from the first record.
func (s *CSVSink) WithHeaders(headers ...string) *CSVSink {
	s.headers = headers
	return s
}

// Comma sets the field delimiter. Default is ','.
func (s *CSVSink) Comma(c rune) *CSVSink {
	s.comma = c
	return s
}

// Open opens the destination file and prepares the CSV writer.
func (s *CSVSink) Open(ctx context.Context) error {
	if s.file != nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	f, err := os.Create(s.path)
	if err != nil {
		return fmt.Errorf("create csv: %w", err)
	}
	s.file = f
	s.writer = csv.NewWriter(f)
	s.writer.Comma = s.comma
	s.wrote = false
	return nil
}

// Write encodes a single record as one CSV row.
func (s *CSVSink) Write(ctx context.Context, r intake.Record) error {
	if s.writer == nil {
		return intake.NewError(intake.ErrKindConfig, "sink_not_open", "CSVSink.Write called before Open")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if !s.wrote {
		if len(s.headers) == 0 {
			s.headers = sortedKeys(r)
		}
		if err := s.writer.Write(s.headers); err != nil {
			return fmt.Errorf("write csv header: %w", err)
		}
		s.wrote = true
	}

	row := make([]string, len(s.headers))
	for i, h := range s.headers {
		row[i] = stringify(h, r[h])
	}
	if err := s.writer.Write(row); err != nil {
		return fmt.Errorf("write csv row: %w", err)
	}
	return nil
}

// Close flushes any buffered rows and closes the file.
func (s *CSVSink) Close() error {
	var firstErr error
	if s.writer != nil {
		s.writer.Flush()
		if err := s.writer.Error(); err != nil {
			firstErr = err
		}
		s.writer = nil
	}
	if s.file != nil {
		if err := s.file.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		s.file = nil
	}
	s.wrote = false
	return firstErr
}

// Headers returns the column order used for output. Available after
// the first call to Write.
func (s *CSVSink) Headers() []string {
	out := make([]string, len(s.headers))
	copy(out, s.headers)
	return out
}

func stringify(key string, v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(t), 'f', -1, 32)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}

func sortedKeys(r intake.Record) []string {
	seen := make(map[string]struct{}, len(r))
	keys := make([]string, 0, len(r))
	for k := range r {
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		keys = append(keys, k)
	}
	// Stable, predictable order. Strings are compared lexicographically.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	return keys
}
