# gh-arrow

A planned GitHub CLI extension to copy, display, and discover files in GitHub repositories and mapped local projects. The command will be `gh arrow`; the extension repository and executable will be named `gh-arrow`.

**Status:** Go scaffold only. The executable supports `--help`; file operations are not implemented and no installable release has been published. The hosted repository is `gh-arrow`.

Planned usage:

```sh
gh arrow cp github:owner:project:path/to/file.yaml ./local.yaml
gh arrow cat local:project2:file.md
gh arrow ls 'github:owner:project:docs/**/*.md'
gh arrow ls --org tools-org --topic cli README.md --json
```

GitHub sources prefer configured local working trees. Other remote operations use GitHub CLI API access and its existing credentials, including accessible private repositories. Operations never prompt. Listings support sorted text or JSON; `cat` preserves raw bytes. The default `./config.yaml` is trusted local configuration: review it before running in an unfamiliar checkout, or select a trusted file with `--config`. See the [configuration example](doc/design-meta/examples/config.yaml).

Read the [generated specification](doc/design/specification.md) for command contracts, requirements, acceptance scenarios, and open questions. The editable source lives in [doc/design-meta](doc/design-meta/AGENTS.md), assembled by [app.cue](doc/design-meta/app.cue). The [original proposal](doc/history/initial-spec.md) is historical.

To update the design, edit its sources and regenerate from the repository root:

```sh
# Requires Python 3 and flyb on PATH.
sh scripts/doc-design.sh
```

Commit both the sources and generated Markdown. AI agents should start with [AGENTS.md](AGENTS.md) and retrieve only the relevant concept records.

Local development requires Go 1.24+, Python 3.11+, Make, GitHub CLI, and `gh flarebyte` v0.24.0 (the version pinned in CI). Install the tooling once, then run checks from the repository root:

```sh
gh extension install flarebyte/gh-flarebyte --pin v0.24.0
make check-tools
make review
make build
```

`make help` lists the targets. `make test` runs the release-tooling tests and CLI smoke checks as well as the Go test command; there are no Go test cases yet. The Go executable uses only the standard library. `make coverage` reports Go coverage separately.

The initial build targets are `darwin-arm64`, `linux-amd64`, and `windows-amd64`. These are scaffold targets, not a release support guarantee. For a local extension installation:

```sh
make build-go
gh extension install .
gh arrow --help
```

CI and release builds use Go 1.27.3; `go.mod` retains Go 1.24 as the minimum language version. Use a supported, patched Go toolchain for local development.

Pull requests and pushes to `main` run formatting, config/CSV validation, lint, tests, smoke checks, and cross-platform builds in [CI](.github/workflows/ci.yml). `make doc-design` additionally requires flyb to regenerate the specification.

Publishing uses the manually dispatched [release workflow](.github/workflows/release.yml). Before the first release, implement the intended commands and replace `0.0.0-dev` in `main.project.yaml` with the chosen release version. Preflight blocks that scaffold version and existing tags/releases. If publishing fails after tag creation, follow the [release recovery procedure](doc/release-recovery.md) before retrying. The workflow verifies the default-branch commit, builds and checksums assets, then publishes those same assets with write permission confined to the publishing job. Prerelease versions are marked as prereleases. `make build-release` stages assets locally in `build/release` and refuses an existing staging directory; use a clean checkout for each release build. It does not publish.

Build version injection and a CLI `--version` remain future work: flarebyte v0.24.0 hardcodes its own metadata package in linker flags. No release has been published by this setup.

The entry point is `cmd/gh-arrow/main.go`. Development and build settings live in `.gh-flarebyte.cue`; `main.project.yaml` holds the development version. Build outputs and Go caches are ignored. Repository sync and release publication are separate actions; local setup does not require either.

The extension follows GitHub's [extension packaging conventions](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions).
