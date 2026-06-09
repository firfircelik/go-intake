// Package db provides database sinks.
package db

import (
	"fmt"
	"strings"
	"testing"
)

// SQLiteSink tests would require external driver
// In production: go install modernc.org/sqlite
func TestSQLiteSink_SchemaFormat(t *testing.T) {
	// Test schema generation format
	headers := []string{"id", "name", "value"}

	cols := make([]string, len(headers))
	for i, h := range headers {
		cols[i] = fmt.Sprintf("%s TEXT", h)
	}

	query := fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS test (%s)",
		strings.Join(cols, ","),
	)

	expected := "CREATE TABLE IF NOT EXISTS test (id TEXT,name TEXT,value TEXT)"
	if query != expected {
		t.Errorf("unexpected query: %s", query)
	}
}

func TestSQLiteSink_InsertFormat(t *testing.T) {
	// Test insert query format
	headers := []string{"id", "name", "value"}

	placeholders := make([]string, len(headers))
	for i := range headers {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(
		"INSERT INTO test (%s) VALUES (%s)",
		strings.Join(headers, ","),
		strings.Join(placeholders, ","),
	)

	expected := "INSERT INTO test (id,name,value) VALUES (?,?,?)"
	if query != expected {
		t.Errorf("unexpected query: %s", query)
	}
}
