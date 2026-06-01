# AGENTS.md

## Project

This repository is a Go-native, library-first ETL/data-intake toolkit.

It helps Go developers turn unknown or messy input data into validated, transformed, record-oriented output.

## Non-goals

Do not implement:
- CLI
- web UI
- scheduler
- DAG engine
- distributed runner
- dataframe abstraction
- Airbyte/Airflow clone
- connector marketplace
- PDF parser
- ML features

## Design principles

- Library-first
- Small public API
- Idiomatic Go
- Interface-based extension
- Streaming record processing
- Explicit errors
- Test everything
- Avoid heavy dependencies
- Keep examples simple and compiling

## Core abstractions

- Record
- Source
- Sink
- Transformer
- Validator
- Pipeline
- Inspector/Profile
- Quarantine sink

## Quality requirements

Before completing any task, run:

```bash
go test ./...
go vet ./...
go test -race ./...