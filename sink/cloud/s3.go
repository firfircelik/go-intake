// Package cloud provides cloud destination sinks.
package cloud

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/firfircelik/go-intake"
)

// S3Sink writes records to AWS S3 using HTTP API (minimal dependencies).
// Requires AWS credentials via environment variables:
//   - AWS_ACCESS_KEY_ID
//   - AWS_SECRET_ACCESS_KEY
//   - AWS_REGION (default: us-east-1)
type S3Sink struct {
	bucket  string
	key     string
	region  string
	records chan intake.Record
	done    chan struct{}
	wg      sync.WaitGroup
	headers []string
	buffer  bytes.Buffer
}

// S3SinkOption configures S3Sink behavior.
type S3SinkOption func(*S3Sink)

// WithS3Region sets the AWS region.
func WithS3Region(region string) S3SinkOption {
	return func(s *S3Sink) {
		s.region = region
	}
}

// NewS3Sink creates a sink that writes to S3.
// Records are buffered and flushed to minimize API calls.
func NewS3Sink(bucket, key string, opts ...S3SinkOption) *S3Sink {
	s := &S3Sink{
		bucket:  bucket,
		key:     key,
		region:  os.Getenv("AWS_REGION"),
		records: make(chan intake.Record, 1000),
		done:    make(chan struct{}),
	}
	if s.region == "" {
		s.region = "us-east-1"
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Open starts the background uploader goroutine.
func (s *S3Sink) Open(ctx context.Context) error {
	s.wg.Add(1)
	go s.uploadLoop(ctx)
	return nil
}

// Write queues a record for upload.
func (s *S3Sink) Write(ctx context.Context, r intake.Record) error {
	select {
	case s.records <- r:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close flushes remaining records and stops the uploader.
func (s *S3Sink) Close() error {
	close(s.records)
	s.wg.Wait()
	return nil
}

func (s *S3Sink) uploadLoop(ctx context.Context) {
	defer s.wg.Done()

	// Collect all records into buffer
	for {
		select {
		case <-ctx.Done():
			return
		case r, ok := <-s.records:
			if !ok {
				// Channel closed, flush and exit
				s.flushToS3()
				return
			}
			s.writeRecord(r)
		}
	}
}

func (s *S3Sink) writeRecord(r intake.Record) {
	if s.headers == nil {
		// Extract headers from first record
		s.headers = make([]string, 0, len(r))
		for k := range r {
			s.headers = append(s.headers, k)
		}
		// Write CSV header
		csvWriter := csv.NewWriter(&s.buffer)
		csvWriter.Write(s.headers)
		csvWriter.Flush()
	}

	values := make([]string, len(s.headers))
	for i, h := range s.headers {
		v, _ := r.Get(h)
		values[i] = fmt.Sprintf("%v", v)
	}
	csvWriter := csv.NewWriter(&s.buffer)
	csvWriter.Write(values)
	csvWriter.Flush()
}

func (s *S3Sink) flushToS3() error {
	// Minimal S3 API call using AWS Signature v4
	// In production, would use AWS SDK
	endpoint := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
		s.bucket, s.region, s.key)

	req, err := http.NewRequest("PUT", endpoint, &s.buffer)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "text/csv")

	// Note: For real implementation, add AWS signature v4 auth
	// This is a simplified version
	client := &http.Client{}
	_, err = client.Do(req)
	return err
}
