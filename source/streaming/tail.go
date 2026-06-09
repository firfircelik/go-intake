// Package streaming provides streaming sources for real-time data ingestion.
package streaming

import (
	"bufio"
	"context"
	"io"
	"os"
	"time"

	"github.com/firfircelik/go-intake"
)

// FileTailSource reads new lines appended to a file, similar to `tail -f`.
// Useful for log monitoring and real-time log processing.
type FileTailSource struct {
	path    string
	file    *os.File
	scanner *bufio.Scanner
	delay   time.Duration
	polling bool
}

// FileTailOption configures FileTailSource behavior.
type FileTailOption func(*FileTailSource)

// WithTailDelay sets the polling interval when file is idle.
func WithTailDelay(d time.Duration) FileTailOption {
	return func(s *FileTailSource) {
		s.delay = d
	}
}

// NewFileTailSource creates a source that tails a file for new lines.
func NewFileTailSource(path string, opts ...FileTailOption) *FileTailSource {
	s := &FileTailSource{
		path:  path,
		delay: 100 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Open opens the file for reading.
func (s *FileTailSource) Open(ctx context.Context) error {
	f, err := os.Open(s.path)
	if err != nil {
		return err
	}
	// Seek to end if following mode (default)
	s.file = f
	s.scanner = bufio.NewScanner(f)
	s.scanner.Buffer(make([]byte, 64*1024), 1024*1024) // 1MB max line
	return nil
}

// Read returns the next line as a record.
func (s *FileTailSource) Read(ctx context.Context) (intake.Record, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if s.scanner.Scan() {
				line := s.scanner.Text()
				// Create record with a single field for the line
				rec := make(intake.Record)
				rec["line"] = line
				return rec, nil
			}
			// Check for errors
			if err := s.scanner.Err(); err != nil && err != io.EOF {
				return nil, err
			}
			// Wait and retry
			time.Sleep(s.delay)
		}
	}
}

// Close closes the file.
func (s *FileTailSource) Close() error {
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}
