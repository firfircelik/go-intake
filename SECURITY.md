# Security

If you discover a security vulnerability in `intake`, please
report it privately to the maintainers rather than opening a
public GitHub issue.

## Reporting

Open a GitHub security advisory at
`github.com/firfircelik/go-intake/security/advisories/new` with:

- A short description of the vulnerability.
- Steps to reproduce, or a proof-of-concept.
- The expected and actual behaviour.
- The version(s) of `intake` affected.

The maintainers will acknowledge the report within seven days and
aim to ship a fix or a documented mitigation in the next minor
or patch release.

## Scope

`intake` is a pure Go library with no third-party dependencies.
The most plausible security concerns are:

- The bundled `source.CSV`, `source.JSONL`, and `quarantine.JSONL`
  read and write files from paths supplied by the caller. If you
  pass untrusted paths you are responsible for validating them.
- The `validate.Regex` validator compiles a regular expression at
  construction time. A pathological pattern can be expensive to
  match against long strings; treat caller-supplied patterns as
  untrusted and consider running `regexp.Match` with a timeout.
- The `discover.InspectSource` function reads up to
  `Options.SampleSize` records. Make sure the source is bounded
  before calling it on an untrusted stream.

Out of scope: any vulnerability in the Go standard library or
in the user's own `Source` / `Sink` / `Quarantine` implementations.

## Disclosure

The maintainers follow a coordinated disclosure model. Please
give them a reasonable window (typically 90 days) to ship a fix
before publishing details.
