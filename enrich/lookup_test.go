// Package enrich provides lookup and enrichment transformers.
package enrich

import (
	"context"
	"testing"

	"github.com/firfircelik/go-intake"
)

func TestCacheLookup_Apply(t *testing.T) {
	lookupCache := map[string]any{
		"1": map[string]any{"name": "Alice", "dept": "Engineering"},
		"2": map[string]any{"name": "Bob", "dept": "Sales"},
	}

	enricher := NewCacheLookup("id", lookupCache)

	rec := make(intake.Record)
	rec["id"] = "1"

	result, err := enricher.Apply(context.Background(), rec)
	if err != nil {
		t.Fatal(err)
	}

	// Check original preserved
	_, ok := result.Get("id")
	if !ok {
		t.Error("original field missing")
	}
}

func TestStaticMapEnrich_Apply(t *testing.T) {
	mapping := map[string]string{
		"US": "United States",
		"UK": "United Kingdom",
	}

	enricher := NewStaticMapEnrich("country_code", "country_name", mapping)

	rec := make(intake.Record)
	rec["country_code"] = "US"

	result, err := enricher.Apply(context.Background(), rec)
	if err != nil {
		t.Fatal(err)
	}

	val, ok := result.Get("country_name")
	if !ok || val != "United States" {
		t.Errorf("expected United States, got %v", val)
	}
}
