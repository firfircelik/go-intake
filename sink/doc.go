// Package sink contains bundled Sink implementations for intake
// pipelines.
//
// A Sink consumes a stream of Records. It is opened once with Open,
// then written to repeatedly with Write, and finally closed with
// Close. Close must be safe to call even when Open was never
// called or failed; the pipeline defers Close before Open so a
// failed Open still tears down the previously-opened components.
//
// The bundled sinks are:
//
//   - CSVSink, built with CSV(path). It writes a header row before
//     the first record, choosing either WithHeaders or the sorted
//     key set of the first record. The delimiter defaults to ','
//     and is configurable with Comma.
//   - JSONLSink, built with JSONL(path). It writes one JSON object
//     per line, using encoding/json with HTML escaping disabled.
package sink
