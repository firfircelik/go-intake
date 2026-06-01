# Contributing

Thanks for your interest in `intake`. The project is small and
the maintainers are responsive; please open an issue before
starting non-trivial work so we can agree on direction first.

## Workflow

1. Fork the repository and create a branch from `main`.
2. Make your change. Keep commits small and focused.
3. Run the quality gates locally before pushing:
   ```sh
   go test -count=1 ./...
   go vet ./...
   gofmt -l .
   go test -race -count=1 ./...
   ```
4. Open a pull request that describes the change and links any
   related issue.
5. Address review feedback by pushing new commits; do not force
   push to your branch after review has started.

## Coding conventions

- Match the style of the surrounding code; run `gofmt` before
  committing.
- Every exported identifier gets a doc comment. Unexported
  identifiers do not need one, but the package-level doc must
  explain what the package is for.
- Do not introduce third-party dependencies. Everything in the
  project uses only the Go standard library.
- Do not add a CLI, a DAG/scheduler, a dataframe abstraction, a
  PDF parser, or a connector marketplace. These are explicit
  non-goals.
- Prefer small, composable APIs over large frameworks.

## Tests

- Tests live next to the code they exercise (`foo.go` and
  `foo_test.go` in the same package).
- Use `t.TempDir()` for filesystem isolation; do not depend on
  the working directory or absolute paths.
- When you add a new public identifier, add a test that exercises
  the happy path, the unhappy path, and the documented edge
  cases.

## Releases

The maintainers tag releases from `main` once the test suite and
the `v0.x` API stability contract allow it. v0.1.x releases may
include breaking changes; v1.0 will freeze the surface.
