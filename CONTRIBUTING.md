# Contributing to gh-arrow

Start with the [README](README.md) for installation and usage,
then the [technical design](TECHNICAL_DESIGN.md) to locate the code being changed.
Run the commands below from the repository root.

## Development setup

You need GitHub CLI (tested with 2.103.3), Go 1.25+, Make, Python 3.11+,
and `gh flarebyte` v0.24.0. The module pins Go 1.27.3; the Go launcher
downloads that toolchain when necessary. Install the tooling once:

```sh
gh extension install flarebyte/gh-flarebyte --pin v0.24.0
make check-tools
make build-go
./gh-arrow --help
```

To install the development extension from this checkout, run `gh extension install .`
after `make build-go`. An existing installed extension must be removed first with
`gh extension remove arrow`.

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
| `make test` | Go tests, Python release-tooling and reserve-sync tests, and executable smoke checks |
| `make test-go` | Configured Go unit and acceptance tests |
| `make test-scripts` | Offline release tooling and reserve-sync recipe tests |
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
version metadata. Python release-tooling and reserve-sync tests live in `scripts/test_*.py`;
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

## Refresh code analysis

The optional [Thoth analysis](thoth-meta/README.md) summarizes Go source and test
files. Its guide lists pipeline relevance, coverage gaps, prerequisites, and
refresh commands. Review generated metadata and preserve curated purpose fields
when refreshing it; analysis generation does not run the tests above.

### Sync canonical configs

The three `pipeline-*.thoth.cue` configs are unchanged copies managed by
`baldrick-reserve`. The [Makefile](Makefile) lists each canonical source, local
destination. `RESERVE_REF` is deliberately empty while the reserve changes await
review, commit, and publication; the targets fail before contacting GitHub until
a full 40-character commit SHA is supplied. Creating or updating sync targets
does not require committing the reserve changes.

After the reserve changes are reviewed and explicitly committed and published,
verify that the revision contains the shared files and set `RESERVE_REF` in the
Makefile. These targets require installed `gh arrow`, Python 3, and network
access to GitHub. A commit available only in a local checkout cannot be
retrieved. From this repository's root, run:

```sh
make reserve-check
make reserve-sync
```

For a one-off revision, pass `RESERVE_REF='<published-full-commit-sha>'` to either
command. The placeholder must be replaced with a verified full SHA. Revision
validation checks syntax locally; successful retrieval confirms availability.

`reserve-check` detects drift without changing the configs. `reserve-sync`
replaces all three listed configs, including the currently inactive TypeScript
e2e config; review local changes before running it. These targets are separate
from build and test workflows. Files are replaced individually, so a later
failure can leave earlier files updated. The dedicated
[Arrow config](.config/reserve-arrow.cue) prevents global local mappings from
overriding the pinned remote sources.

For improvements, edit the canonical files on the reserve's active branch,
validate their consumers, and leave changes uncommitted for review. If the active
branch is `main` or `master`, ask how to proceed before editing. Commit or push
only when explicitly requested. Once a revision containing the changes is
published, update `RESERVE_REF` here and review the sync diff. The reserve's
`shared/CATALOG.md` records compatibility and runtime prerequisites.

`scripts/test_reserve_sync.py` checks these Makefile recipes with an offline
Arrow substitute as part of `make test-scripts`. It covers drift, missing files,
fetch failures, symlink rejection during checks, repeated sync, and rejection of
pending or malformed revisions before Arrow runs. It does not
verify that the pinned revision is reachable on GitHub.

## Prepare a release

Follow [Publishing a release](doc/releasing.md) for version changes and
publication. Repository sync and publishing are separate from development checks.
