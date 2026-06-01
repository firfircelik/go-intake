// inspect_csv is a discover-package example.
//
// It opens a CSV file, samples the first N records, infers a
// DatasetProfile, and prints the inferred field types plus any
// data-quality issues that were found.
//
// Usage:
//
//	go run ./examples/inspect_csv input.csv [sample-size]
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/firfircelik/go-intake/discover"
	"github.com/firfircelik/go-intake/source"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: inspect_csv <input.csv> [sample-size]")
		os.Exit(2)
	}
	path := os.Args[1]
	opts := discover.Options{}
	if len(os.Args) >= 3 {
		n, err := strconv.Atoi(os.Args[2])
		if err != nil || n <= 0 {
			fmt.Fprintln(os.Stderr, "sample-size must be a positive integer")
			os.Exit(2)
		}
		opts.SampleSize = n
	}

	profile, err := discover.InspectSource(context.Background(), source.CSV(path), opts)
	if err != nil {
		log.Fatalf("inspect: %v", err)
	}
	fmt.Printf("Sampled %d records (%d fields) from %s\n\n",
		profile.RowCount, profile.FieldCount, path)
	for _, f := range profile.Fields {
		fmt.Printf("%-20s %-8s confidence=%.2f null=%d/%d unique=%d",
			f.Name, f.Type, f.TypeConfidence, f.NullCount, f.NonNullCount, f.UniqueCount)
		if f.Type == discover.TypeString {
			fmt.Printf(" len=[%d..%d]", f.MinLength, f.MaxLength)
		}
		if f.Type == discover.TypeInt || f.Type == discover.TypeFloat {
			fmt.Printf(" range=[%v..%v]", f.MinFloat, f.MaxFloat)
		}
		if len(f.Examples) > 0 {
			fmt.Printf(" examples=%v", truncate(f.Examples, 3))
		}
		fmt.Println()
	}
	if len(profile.Issues) > 0 {
		fmt.Println()
		fmt.Println("Issues:")
		for _, i := range profile.Issues {
			if i.Field == "" {
				fmt.Printf("  [%s] %s: %s\n", i.Severity, i.Code, i.Message)
			} else {
				fmt.Printf("  [%s] %s on %q: %s\n", i.Severity, i.Code, i.Field, i.Message)
			}
		}
	}
}

func truncate(s []string, n int) []string {
	if len(s) <= n {
		return s
	}
	out := make([]string, 0, n+1)
	out = append(out, s[:n]...)
	out = append(out, "...")
	return out
}
