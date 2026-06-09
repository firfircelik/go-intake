# go-intake

A Go-native, **library-first** toolkit for turning unknown or
messy input data into validated, transformed, record-oriented
output. Tiny public surface, no third-party dependencies, no CLI.

```go
p := intake.New().
    From(source.CSV("input.csv")).
    Transform(
        transform.NormalizeHeaders(transform.SnakeCase),
        transform.TrimStrings(),
        transform.ParseFloat("price"),
    ).
    Validate(
        validate.Required("product"),
        validate.Min("price", 0),
    ).
    OnInvalid(quarantine.JSONL("bad-records.jsonl")).
    To(sink.JSONL("output.jsonl"))

if err := p.Run(context.Background()); err != nil {
    log.Fatal(err)
}
fmt.Printf("%+v\n", p.Stats())
```

## Why record-oriented data intake

Most real-world pipelines start with a flat file, a JSON stream,
or a CSV dump and end with the same shape after cleaning. In
between there is normalisation, parsing, validation, and the
occasional quarantine of bad rows. That work is small in any
single project, but it is the same work every time.

`go-intake` is a small library that codifies that work. It gives
you:

- A handful of small, composable interfaces (`Source`, `Sink`,
  `Transformer`, `Validator`, `Quarantine`).
- A fluent `Pipeline` builder that wires them together.
- A streaming model: only the record currently in flight lives
  in memory, so the library is happy with a 1 GB CSV and a 1 TB
  CSV equally.
- A `discover` sub-package for sampling and schema inference,
  useful for understanding a dataset before wiring the real
  pipeline.

It is not a framework. You bring your own `Source` and `Sink`
implementations when the bundled ones are not enough, and the
pipeline wires them together without ceremony.

## What it is

- **A library**, not a service. You call it from your Go code.
- **Streaming.** The whole dataset is never loaded into memory;
  only the record currently in flight is.
- **Composable.** Five interfaces, a fluent builder, and a
  stable set of bundled primitives for common cases.
- **Honest about validation.** A pipeline can collect every
  failing validator per record and send one `InvalidRecord`
  event downstream, so a single bad row never hides another.
- **Zero dependencies.** Pure standard library. Go 1.23+.

## What it is not

- **Not** a CLI. There is no `intake` binary; you embed the
  library.
- **Not** a DAG engine, scheduler, or distributed runner. One
  process, one pipeline, no retries across stages.
- **Not** a dataframe library. Records are `map[string]any`;
  there is no columnar type, no row index, no in-place mutation
  API, and no SQL.
- **Not** a connector marketplace. Sources and sinks are small
  interfaces; you implement the one or two you need.
- **Not** a PDF parser, an OCR tool, an ML feature store, an
  Airbyte/Airflow clone, or a YAML-driven orchestrator. See
  *Non-goals* at the end of this file.

## Installation

```sh
go get github.com/firfircelik/go-intake
```

Requires Go 1.23 or newer. No third-party dependencies.

## Concepts

| Type | Role |
|---|---|
| `intake.Record` | `map[string]any` — one row of data. |
| `intake.Source` | `Open` / `Read` (returns `io.EOF` at end) / `Close`. |
| `intake.Sink` | `Open` / `Write` / `Close`. |
| `intake.Transformer` | `Apply(ctx, r) (Record, error)`. Never mutates `r`; returns a fresh record on every call. A non-nil error sends the record to quarantine. |
| `intake.Validator` | `Validate(ctx, r) error`. Read-only. A non-nil error sends the record to quarantine. |
| `intake.Quarantine` | `Open` / `Write(ctx, InvalidRecord)` / `Close` — destination for rejected records. |
| `intake.Pipeline` | Wires the above together and runs them. |
| `intake.Stats` | `Read`, `Written`, `Invalid`, `Failed` counters. |

A `Pipeline` is single-use. Build a new one for each run.

## Examples

The four examples below cover the realistic use cases. The full
runtable versions live under `examples/`.

### 1. Basic pipeline

Read a CSV, normalise headers, trim strings, parse `price` as a
float, require `product`, and reject negative prices.

```go
p := intake.New().
    From(source.CSV("input.csv")).
    Transform(
        transform.NormalizeHeaders(transform.SnakeCase),
        transform.TrimStrings(),
        transform.ParseFloat("price"),
    ).
    Validate(
        validate.Required("product"),
        validate.Min("price", 0),
    ).
    OnInvalid(quarantine.JSONL("bad-records.jsonl")).
    To(sink.JSONL("output.jsonl"))

if err := p.Run(context.Background()); err != nil {
    log.Fatal(err)
}
fmt.Printf("%+v\n", p.Stats())
```

### 2. Inspect a file before building the pipeline

Schema inference is **best-effort**. Use `discover.InspectSource`
to see roughly what the data looks like, then wire the real
pipeline with explicit `validate.*` rules for the columns that
matter.

```go
src := source.CSV("dirty.csv")
profile, err := discover.InspectSource(ctx, src, discover.Options{
    SampleSize:               1000,
    NullIssueThreshold:       0.5,
    MixedTypeIssueThreshold:  0.8,
})
if err != nil {
    log.Fatal(err)
}
for _, f := range profile.Fields {
    fmt.Printf("%s: %s (confidence %.2f)\n", f.Name, f.Type, f.TypeConfidence)
}
for _, i := range profile.Issues {
    fmt.Printf("[%s] %s on %q: %s\n", i.Severity, i.Code, i.Field, i.Message)
}
```

The returned profile is a `DatasetProfile` with inferred
`FieldType` per field, null / unique / range statistics, and a
list of `Issue` entries (`high_null_ratio`, `mixed_types`,
`empty_field_name`, `no_records_sampled`). See *Discover* below
for the full surface.

### 3. Validation + quarantine

If multiple validators reject the same record, the pipeline
collects every error and sends **one** invalid-record event to
the quarantine. The collected errors are also available as an
`*intake.MultiError` whose `Unwrap() []error` lets `errors.As`
recover each one, and whose `Is(target error) bool` lets
`errors.Is` search the collected slice.

```go
if err := p.Run(ctx); err != nil {
    var me *intake.MultiError
    if errors.As(err, &me) {
        for _, e := range me.Errors {
            // handle e individually
        }
    }
    if errors.Is(err, mySentinel) { /* walks every collected error */ }
}
```

The bundled `quarantine.JSONL` serialises each rejected record
as the original record fields plus three reserved keys:

- `_errors` (`intake.QuarantineKeyErrors`) — a list of error
  objects. Each `*validate.ValidationError` becomes
  `{field, rule, message, value}`; other errors become
  `{message}`.
- `_stage` (`intake.QuarantineKeyStage`) — `transform` or
  `validation`.
- `_timestamp` (`intake.QuarantineKeyTimestamp`) — RFC 3339
  timestamp captured at quarantine time (UTC, with nanoseconds).

These three keys are reserved; if they already exist on the
input record, the quarantine sink overwrites them. Custom
`Quarantine` implementations that want to stay interchangeable
with consumers of `quarantine.JSONL` should reserve the same
three keys.

### 4. Custom Source / Sink

The bundled `Source` and `Sink` types cover CSV and JSONL.
Anything else is a one-method interface:

```go
type upperValidator struct{}

func (upperValidator) Validate(_ context.Context, r intake.Record) error {
    for _, v := range r {
        if s, ok := v.(string); ok && s != strings.ToUpper(s) {
            return errors.New("must be uppercase")
        }
    }
    return nil
}

p := intake.New().
    From(source.CSV("in.csv")).
    Validate(upperValidator{}).
    To(sink.JSONL("out.jsonl"))
```

For a custom `Source` (e.g. an HTTP endpoint, a database
reader, a generated test stream) implement the three-method
`Source` interface and pass the value to `From`. The pipeline
calls `Open` once, `Read` until it returns `io.EOF`, and
`Close` on every exit path including errors.

## Error behaviour

- `io.EOF` from `Source.Read` ends the pipeline cleanly.
- A transformer error increments `Stats.Invalid` and forwards
  the original (pre-transform) record to the quarantine (if
  configured). Without a quarantine, invalid records are
  skipped.
- If multiple validators reject the same record, the pipeline
  collects every error and sends **one** invalid-record event to
  the quarantine. `Stats.Invalid` increments by one per record,
  regardless of how many validators fired.
- A source open/read error or a sink open/write error is fatal
  and increments `Stats.Failed`. The run is aborted and the
  error is returned to the caller, wrapped in an `*intake.Error`
  with a stable `Code` and an `ErrorKind` (`source`, `transform`,
  `validation`, `sink`, `quarantine`, `config`).

## Bundled sources and sinks

| Source | Constructor | Notes |
|---|---|---|
| CSV | `source.CSV(path)` / `source.NewCSVSource(r)` | RFC 4180 CSV with a header row. `Comma(rune)`, `SkipEmptyLines(bool)`. |
| JSONL | `source.JSONL(path)` / `source.NewJSONLSource(r)` | Newline-delimited JSON. Each non-empty line is decoded as a `Record`. |
| **FileTail** | `streaming.NewFileTailSource(path)` | Tails files for new lines (log monitoring). `WithTailDelay(d)`. |

| Sink | Constructor | Notes |
|---|---|---|
| CSV | `sink.CSV(path)` | `WithHeaders(...)` to control column order; `Comma(rune)`. |
| JSONL | `sink.JSONL(path)` | One JSON object per line. |
| **S3** | `cloud.NewS3Sink(bucket, key)` | AWS S3 upload. `WithS3Region(region)`. |
| **SQLite** | `db.NewSQLiteSink(dsn, table)` | SQLite database. Auto-creates table. |

| Quarantine | Constructor | Notes |
|---|---|---|
| JSONL | `quarantine.JSONL(path)` | Spreads the record fields and adds `_errors`, `_stage`, `_timestamp`. |

## Bundled transforms

| Function | Purpose |
|---|---|
| `transform.NormalizeHeaders(style)` | Rewrite every key using a `HeaderStyle`. Built-in: `SnakeCase`, `CamelCase`, `KebabCase`, `LowerCase`, `UpperCase`. |
| `transform.TrimStrings(fields...)` | Trim whitespace; defaults to all string fields. |
| `transform.LowerStrings(fields...)` / `UpperStrings` | Lowercase / uppercase string values. |
| `transform.ParseFloat(field)` | Parse a string field to `float64`. |
| `transform.ParseInt(field)` | Parse to `int64`. |
| `transform.ParseBool(field)` | Parse a recognised bool string. |
| `transform.ParseDate(field, layouts...)` | Parse a string field to `time.Time` using the supplied `time.Parse` layouts (tried in order). |
| `transform.Rename(from, to)` / `Copy(from, to)` | Reshape the record's keys. |
| `transform.Drop(fields...)` / `Keep(fields...)` | Remove or restrict fields. |
| `transform.AddField(field, value)` | Set a field to a constant value. |
| `transform.MapField(field, fn)` | Apply a user function `func(any) (any, error)` to a field's value. |
| **enrich.CacheLookup** | Enrich records with cached lookup values. |
| **enrich.StaticMapEnrich** | Map field values to static values. |

**Contract:** every transformer in this package never mutates
the input record. `Apply` always returns a fresh `Record`, even
on no-op, on error, and when the field is missing. Mutating the
returned record does not affect any other record in the
pipeline.

## Bundled validators

Core composable set:

| Function | Purpose |
|---|---|
| `validate.Required(field)` | Field must be present and non-empty. |
| `validate.Min(field, n)` / `Max` | Numeric `>=` n / `<=` n. |
| `validate.Between(field, lo, hi)` | Numeric in `[lo, hi]`. |
| `validate.Regex(field, pattern)` | Regular-expression match. |
| `validate.Enum(field, values...)` | String is one of the allowed values. |
| `validate.NotFuture(field)` | `time.Time` value is at or before now. |
| `validate.Custom(name, fn)` | `fn func(Record) error` predicate; `name` becomes the `Rule` on the returned `ValidationError`. |

Thin convenience wrappers (use only when they read more clearly
than the core primitives):

| Function | Purpose |
|---|---|
| `validate.Present(field)` | Field is present (zero values OK). |
| `validate.Forbidden(field)` | Field must be missing or empty. |
| `validate.ExclusiveRange(field, lo, hi)` | Numeric in `(lo, hi)`. |
| `validate.MinLen(field, n)` / `MaxLen` / `Len` | String length in runes. |
| `validate.Email(field)` | `Regex` with a permissive email pattern. |
| `validate.URL(field)` | `net/url` parse + scheme/host check. |

Every failing validator returns a `*validate.ValidationError`
with `Field`, `Rule`, `Message`, and `Value` populated. Use
`errors.As(err, &ve)` to extract it. The `Value` field carries
the actual record value that triggered the failure so
downstream tools can render rich diagnostics without re-reading
the record.

## Discover

`discover.InspectSource` samples an `intake.Source` and returns
a `DatasetProfile` with per-field inference, statistics, and a
list of data-quality issues. Memory use is bounded by
`Options.SampleSize` and `Options.MaxUnique`.

### Inferred types

| `FieldType` | Rule |
|---|---|
| `unknown` | The field had no non-null values. |
| `string` | Fallback. Also the result when a field is mixed (with reduced `TypeConfidence` and a `mixed_types` issue). |
| `int` | Every non-null value parses as an integer (JSON int, or a string parseable by `strconv.ParseInt`). |
| `float` | Every non-null value is a number and at least one is fractional. |
| `bool` | Every non-null value is a JSON bool or a recognised bool string (`true`/`false`/`yes`/`no`/`y`/`n`/`t`/`f`, case-insensitive). The strings `"1"` and `"0"` are intentionally classified as int, not bool. |
| `date` | Every non-null value parses as a date with no time component (`2006-01-02`, `02/01/2006`, `01/02/2006`, `2006/01/02`). |
| `datetime` | Every non-null value parses as a date with a time component (`time.RFC3339`, `2006-01-02 15:04:05`, `2006-01-02T15:04:05`). |

### Issues

`DatasetProfile.Issues` lists every quality problem found. Codes
are stable constants; severities are `"info"`, `"warning"`, or
`"error"`.

| Code | Severity | When |
|---|---|---|
| `no_records_sampled` | info | The source produced no records within `Options.SampleSize` reads. |
| `empty_field_name` | error | At least one record contributed an empty-string key. |
| `high_null_ratio` | warning | A field's `NullRatio` exceeds `Options.NullIssueThreshold` (default 0.5). |
| `mixed_types` | warning | A field's `TypeConfidence` is below `Options.MixedTypeIssueThreshold` (default 0.8). |
| `duplicate_field_name` | — | Reserved for future use; no bundled source currently surfaces it. |

> Inference is best-effort. A field that looks like a date today
> may stop looking like a date after a schema change next
> quarter. Always wire the columns you care about through
> `validate.*` in the real pipeline.

## Examples in this repo

Two runnable examples are provided:

- `examples/csv_to_jsonl` — the full pipeline above. A small
  `sample.csv` is included; the example assumes a `product`
  column (e.g. header `Product` is normalized to `product` by
  the `SnakeCase` style).
- `examples/inspect_csv` — schema + profile a CSV file. A
  `messy.csv` with numeric strings, missing values, an invalid
  date, and mixed types is included.

```sh
go run ./examples/csv_to_jsonl examples/csv_to_jsonl/sample.csv out.jsonl bad.jsonl
go run ./examples/inspect_csv examples/inspect_csv/messy.csv
```

## API stability

This is a `v0.x` library. The surface may change in
`v0.1.x`, `v0.2.x`, etc. The maintainers will document every
breaking change in `CHANGELOG.md` and will avoid gratuitous
churn between minor versions. The v1.0 surface will be frozen
on the contracts below.

- **`intake.Transformer`**: never mutates the input record;
  always returns a fresh `Record` on no-op, on error, and when
  the target field is missing. All `transform.*` transformers
  honour this.
- **`intake.Validator`**: read-only. A non-nil error sends the
  record to quarantine. All `validate.*` validators honour
  this.
- **`intake.Quarantine.Write`**: receives a structured
  `InvalidRecord`. The pipeline populates `Stage` and
  `Timestamp`; the consumer is responsible only for the
  `Record` and `Errors`. The bundled `quarantine.JSONL` reserves
  the three keys listed in *Validation + quarantine* above;
  custom sinks that want to remain interchangeable should
  reserve the same three.
- **`intake.MultiError`**: implements `Unwrap() []error` and
  `Is(target error) bool` so `errors.As` and `errors.Is` work
  across the collected slice. The slice returned by `Unwrap` is
  a fresh copy.
- **Validator split**: 8 core composable validators
  (`Required`, `Min`, `Max`, `Between`, `Regex`, `Enum`,
  `NotFuture`, `Custom`) and 8 thin convenience wrappers
  (`Present`, `Forbidden`, `ExclusiveRange`, `MinLen`, `MaxLen`,
  `Len`, `Email`, `URL`). The wrappers do not add behaviour that
  the core set cannot express.

## Roadmap

The roadmap is short and conservative. v0.1 ships the surface
above. Future minor versions may add:

- More bundled sources and sinks (Parquet, NDJSON over HTTP
  via a caller-supplied `io.Reader`, gzip-aware CSV).
- Optional, opt-in helpers for common transforms (e.g.
  `transform.Money` for currency strings, `transform.PhoneE164`).
- A `validate.Set` and `validate.Map` for record-level rules
  built on `validate.Custom`.
- A `discover.Compact` formatter for the `DatasetProfile`,
  useful in CI logs.

Nothing on this list is committed; see the open issues for
current proposals.

## Non-goals

- A CLI, REPL, or daemon. The library is invoked from your Go
  code.
- A DAG engine, scheduler, retry policy, or distributed runner.
- A dataframe abstraction (no columns, no row index, no
  in-place mutation).
- Database connectors, HTTP sources, or any network-protocol
  primitive in the bundled packages.
- A YAML, TOML, or JSON config layer. Pipelines are built in Go
  so that static analysis and refactors work.
- A connector marketplace or plugin loader.
- A PDF parser, OCR frontend, ML feature store, or anything
  that would require a non-stdlib dependency.

## Example: Log monitoring pipeline

Real-time log processing with streaming source:

```go
// Monitor application logs in real-time
p := intake.New().
    From(streaming.NewFileTailSource("/var/log/app.log")).
    Transform(enrich.NewStaticMapEnrich("level", "level_name", map[string]string{
        "ERROR": "Critical",
        "WARN":  "Warning",
    })).
    Validate(validate.Required("line")).
    To(sink.JSONL("processed_logs.jsonl"))
```

## Example: Database export pipeline

Export validated data to database:

```go
// Export to SQLite
db, _ := db.NewSQLiteSink("file:data.db", "users")
p := intake.New().
    From(source.CSV("users.csv")).
    Transform(transform.ParseInt("age")).
    Validate(validate.Min("age", 18)).
    To(db)
```

## Example: Observability

Monitor pipeline metrics via Prometheus endpoint:

```go
// Prometheus metrics - use as http.Handler
exporter := metrics.NewPrometheusExporter("")
collector := metrics.NewMetricsCollector(exporter)
collector.Collect(func() intake.Stats {
    return pipeline.Stats()
})

// Integrate with your HTTP server
http.Handle("/metrics", exporter)
go http.ListenAndServe(":8080", nil)
```

## Quality gates

```sh
go test -count=1 ./...
go vet ./...
gofmt -l .
go test -race -count=1 ./...
```

The project targets 100% test coverage on the public surface.
Tests use `t.TempDir()` for filesystem isolation and run
cleanly under the race detector.

## License

MIT. See `LICENSE`.
