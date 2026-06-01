// Package source contains bundled Source implementations for
// intake pipelines.
//
// A Source produces a stream of Records. It is opened once with
// Open, then read repeatedly with Read until it returns io.EOF,
// which signals the natural end of the stream. Close must be safe
// to call even when Open was never called or failed; the pipeline
// defers Close before Open so a failed Open still tears down the
// previously-opened components.
//
// The bundled sources are:
//
//   - CSVSource, built with CSV(path) or NewCSVSource(r). It reads
//     RFC 4180-style CSV with a header row. The delimiter defaults
//     to ',' and is configurable with Comma. By default fully-empty
//     rows are skipped; toggle that with SkipEmptyLines.
//   - JSONLSource, built with JSONL(path) or NewJSONLSource(r). It
//     reads newline-delimited JSON, decoding each non-empty line
//     into a Record.
package source
