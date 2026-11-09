# Contributing to gh-arrow

Start with the [README](README.md) for usage and local extension installation,
then the [technical design](TECHNICAL_DESIGN.md) to locate the code being changed.
Run the commands below from the repository root.

## Development setup

Use the Go, Make, and GitHub CLI prerequisites in the
[README quick start](README.md#try-it-from-a-checkout). Development checks also
require Python 3.11+ and `gh flarebyte` v0.24.0. Install the tooling once:

```sh
gh extension install flarebyte/gh-flarebyte --pin v0.24.0
make check-tools
make build-go
./gh-arrow --help
```

`make build-go` produces the executable at the repository root. `make build`
creates the configured platform artifacts and checksums under `build/`.
`.gh-flarebyte.cue` owns development/build settings. Build outputs and local Go
caches are ignored; do not commit them. Local checks do not sync repository
settings or publish releases.

## Check a change

```sh
make review
make test-race
make build
```

`make review` finishes formatting before running lint, tests, and specification
CSV validation, including when Make runs in parallel. `make help` lists targets.

| Command | What it checks |
| --- | --- |
| `make test` | Go tests, Python release-tooling tests, and executable smoke checks |
| `make test-go` | Configured Go unit and acceptance tests |
| `make test-scripts` | Offline release preflight and artifact-staging tests |
| `make smoke` | Builds the executable and exercises local command behavior |
| `make lint` | Flarebyte config validation and Go static checks |
| `make test-race` | Go tests with the race detector |
| `make coverage` | Go coverage report |
| `make sec` | Pinned vulnerability checker; requires downloads/advisory access |
| `make build` | Cross-platform artifacts and checksums |

[CI](.github/workflows/ci.yml) also verifies that formatting leaves tracked Go
files unchanged. [Native checks](.github/workflows/native.yml) run tests, smoke
checks, and vulnerability scans on Linux and macOS. Windows checks are limited
to regressions linked to confirmed implementation bugs, currently PR #3's root
listing failures. Windows artifacts remain cross-built; this does not establish
full native Windows validation.

## Write tests first

Add a failing regression or acceptance test before changing behavior. Reference
the stable scenario IDs in the [specification](doc/design/specification.md).
Keep code modular, prefer small concrete packages, and define interfaces at
consumer boundaries when they make testing or backend reuse useful.

Go tests live beside the code in `internal/*` and at the root for embedded
version metadata. Python release-tooling tests live in `scripts/test_*.py`;
`scripts/smoke.py` exercises the built executable. For a focused Go run with the
same repository-local caches:

```sh
GOCACHE="$PWD/.gocache" GOMODCACHE="$PWD/.gomodcache" go test ./internal/cli
```

Source parsing and glob matching also have native Go fuzz targets. See the
[verification strategy](TECHNICAL_DESIGN.md#verification) for coverage boundaries.
Normal tests use temporary files, fake credentials, and HTTP fixtures; they must
not depend on real credentials or GitHub access. Keep real private-repository
checks opt-in and separate from ordinary tests.

## Update the specification and docs

Use [doc/design-meta/AGENTS.md](doc/design-meta/AGENTS.md) to find the relevant
concept records. Edit the CSV/CUE/example sources in `doc/design-meta`; preserve
rule, decision, command, and scenario IDs. Then regenerate:

```sh
# Requires flyb on PATH in addition to Python 3.
make doc-design
```

This validates CSV structure and references before flyb generation. Commit both
source changes and `doc/design/specification.md`. Do not edit generated Markdown
directly or rewrite the historical proposal. `make validate-design` runs the CSV
checks without regeneration.

Keep usage in the README, contributor workflows here, and implementation
boundaries in the technical design. Link between documents instead of repeating
contracts. [AGENTS.md](AGENTS.md) contains the repository's agent instructions;
the [documentation index](doc/README.md) identifies each document's role.

## Prepare a release

Follow [Publishing a release](doc/releasing.md) for version changes and
publication. Repository sync and publishing are separate from development checks.
