// Package db provides database sinks.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/firfircelik/go-intake"
)

// SQLiteSink writes records to SQLite database.
// No external dependencies - uses stdlib database/sql with modernc/sqlite driver.
type SQLiteSink struct {
	db       *sql.DB
	table    string
	headers  []string
	inserted uint64
}

// NewSQLiteSink creates a sink for SQLite database.
// Requires modernc/sqlite driver: go install modern-csqlite.org/driver
func NewSQLiteSink(dsn, table string) (*SQLiteSink, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	return &SQLiteSink{db: db, table: table}, nil
}

// Open prepares the database.
func (s *SQLiteSink) Open(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Write inserts a record.
func (s *SQLiteSink) Write(ctx context.Context, r intake.Record) error {
	if s.headers == nil {
		// Extract headers from first record
		s.headers = make([]string, 0, len(r))
		for k := range r {
			s.headers = append(s.headers, k)
		}
		// Create table
		return s.createTable()
	}

	// Build insert
	placeholders := make([]string, len(s.headers))
	values := make([]any, len(s.headers))
	for i, h := range s.headers {
		placeholders[i] = "?"
		v, _ := r.Get(h)
		values[i] = v
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		s.table,
		strings.Join(s.headers, ","),
		strings.Join(placeholders, ","),
	)

	_, err := s.db.ExecContext(ctx, query, values...)
	if err == nil {
		s.inserted++
	}
	return err
}

// Close closes the database.
func (s *SQLiteSink) Close() error {
	return s.db.Close()
}

func (s *SQLiteSink) createTable() error {
	// Build CREATE TABLE with inferred types
	cols := make([]string, len(s.headers))
	for i, h := range s.headers {
		cols[i] = fmt.Sprintf("%s TEXT", h)
	}
	query := fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (%s)",
		s.table,
		strings.Join(cols, ","),
	)
	_, err := s.db.Exec(query)
	return err
}
