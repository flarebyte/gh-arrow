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

Local development requires Go 1.24 or newer, GitHub CLI, and the `gh flarebyte` extension. From the repository root:

```sh
gh flarebyte config validate
gh flarebyte format
gh flarebyte lint
gh flarebyte test
gh flarebyte build
./build/gh-arrow-darwin-arm64 --help
```

Choose the binary matching your platform; the initial build targets are `darwin-arm64`, `linux-amd64`, and `windows-amd64`. These are scaffold targets, not a release support guarantee. There are no tests yet. The project currently uses only the Go standard library.

For a local extension installation, build the root executable first:

```sh
GOCACHE="$PWD/.gocache" GOMODCACHE="$PWD/.gomodcache" go build -o gh-arrow ./cmd/gh-arrow
gh extension install .
gh arrow --help
```

The entry point is `cmd/gh-arrow/main.go`. Development and build settings live in `.gh-flarebyte.cue`; `main.project.yaml` holds the development version. Build outputs and Go caches are ignored. Repository sync and release publication are separate actions; local setup does not require either.

The extension follows GitHub's [extension packaging conventions](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions).
