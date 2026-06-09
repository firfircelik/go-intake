package intake_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/firfircelik/go-intake/source"
)

func TestNestedJSONPayloads(t *testing.T) {
	// Create deeply nested JSON data
	nestedData := map[string]any{
		"user": map[string]any{
			"id": 123,
			"address": map[string]any{
				"street": "Main St",
				"city":   "NYC",
				"country": map[string]any{
					"code": "US",
					"name": "United States",
				},
			},
		},
		"metadata": map[string]any{
			"tags":   []any{"tag1", "tag2", "tag3"},
			"counts": []any{1, 2, 3},
		},
	}

	tmpDir := t.TempDir()
	jsonlFile := filepath.Join(tmpDir, "nested.jsonl")

	// Write nested JSON
	f, _ := os.Create(jsonlFile)
	enc, _ := json.Marshal(nestedData)
	f.Write(append(enc, '\n'))
	f.Close()

	// Read with go-intake
	src := source.JSONL(jsonlFile)
	src.Open(context.Background())

	rec, err := src.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Demonstrate nested access
	user, _ := rec.Get("user")
	if user == nil {
		t.Fatal("expected user field")
	}

	// Nested map[string]any access
	userMap := user.(map[string]any)
	address, _ := userMap["address"]
	t.Logf("Address type: %T", address)

	// Deeper nesting
	addressMap := address.(map[string]any)
	country, _ := addressMap["country"]
	countryMap := country.(map[string]any)

	if countryMap["code"] != "US" {
		t.Errorf("expected US, got %v", countryMap["code"])
	}

	// Array access
	tags, _ := rec.Get("metadata")
	metaMap := tags.(map[string]any)
	tagList := metaMap["tags"].([]any)

	if len(tagList) != 3 {
		t.Errorf("expected 3 tags, got %d", len(tagList))
	}

	src.Close()
}
