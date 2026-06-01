# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-06-01

First public release. Library is feature-complete for the v0.1 scope
described in the README. The public surface is small, the contracts
are explicit, and there are no third-party dependencies.

### Added

- `intake` package: the four core interfaces (`Source`, `Sink`,
  `Transformer`, `Validator`) plus `Quarantine` for rejected
  records, and a fluent `Pipeline` builder.
- `transform` package: header normalization, string
  casing/trimming, parsers (`ParseFloat`, `ParseInt`, `ParseBool`,
  `ParseDate`), and field reshaping (`Rename`, `Copy`, `Drop`,
  `Keep`, `AddField`, `MapField`). All transformers are
  non-mutating.
- `validate` package: eight core validators (`Required`, `Min`,
  `Max`, `Between`, `Regex`, `Enum`, `NotFuture`, `Custom`) and
  eight thin convenience wrappers (`Present`, `Forbidden`,
  `ExclusiveRange`, `MinLen`, `MaxLen`, `Len`, `Email`, `URL`).
- `source` package: `CSVSource` and `JSONLSource`.
- `sink` package: `CSVSink` and `JSONLSink`.
- `quarantine` package: `JSONL` quarantine sink that writes the
  original record plus the three reserved metadata keys
  (`_errors`, `_stage`, `_timestamp`).
- `discover` package: `InspectSource` returns a `DatasetProfile`
  with inferred `FieldType` per field, statistics, and a list of
  data-quality `Issue`s.
- `intake.MultiError` and `intake.InvalidRecord` for surfacing
  the structured reason a record was rejected. `MultiError`
  supports `errors.As` and `errors.Is`.
- Two runnable examples: `examples/csv_to_jsonl` and
  `examples/inspect_csv`.

### Notes

- No third-party dependencies. Pure standard library, Go 1.23+.
- Streaming, bounded memory regardless of dataset size.
- Transformers never mutate the input record. Validators never
  mutate the input record. The pipeline collects every failing
  validator per record into a single `InvalidRecord` event so
  downstream consumers see all reasons at once.
- Schema inference in `discover` is best-effort. Important
  fields should always be validated explicitly with `validate.*`
  in a real pipeline.
