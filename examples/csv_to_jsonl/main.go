// csv_to_jsonl is a minimal intake pipeline example.
//
// It reads a CSV file, normalizes headers to snake_case, trims
// string values, parses "price" as a float, requires "product",
// and rejects negative prices to a quarantine file. Valid records
// are written to a JSONL file.
//
// The input CSV must contain a "product" column (the header
// "Product" is normalized to "product" by the snake_case style).
// A small sample input is included in sample.csv.
//
// Usage:
//
//	go run ./examples/csv_to_jsonl input.csv output.jsonl bad-records.jsonl
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/firfircelik/go-intake"
	"github.com/firfircelik/go-intake/quarantine"
	"github.com/firfircelik/go-intake/sink"
	"github.com/firfircelik/go-intake/source"
	"github.com/firfircelik/go-intake/transform"
	"github.com/firfircelik/go-intake/validate"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: csv_to_jsonl <input.csv> <output.jsonl> <bad-records.jsonl>")
		os.Exit(2)
	}
	inPath, outPath, badPath := os.Args[1], os.Args[2], os.Args[3]

	p := intake.New().
		From(source.CSV(inPath)).
		Transform(
			transform.NormalizeHeaders(transform.SnakeCase),
			transform.TrimStrings(),
			transform.ParseFloat("price"),
		).
		Validate(
			validate.Required("product"),
			validate.Min("price", 0),
		).
		OnInvalid(quarantine.JSONL(badPath)).
		To(sink.JSONL(outPath))

	if err := p.Run(context.Background()); err != nil {
		log.Fatalf("pipeline: %v", err)
	}
	s := p.Stats()
	fmt.Printf("read=%d written=%d invalid=%d failed=%d\n",
		s.Read, s.Written, s.Invalid, s.Failed)
}
