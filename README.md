# gh-arrow

A Go GitHub CLI extension to copy, display, and discover files in GitHub repositories and mapped local projects. Invoke it as `gh arrow`.

**Version:** `0.1.0-alpha.1`, an unreleased alpha candidate. No installable release has been published. The commands are implemented and covered by offline tests; native platform checks are configured in CI.

## Try it from a checkout

You need GitHub CLI (tested with 2.103.3), Go 1.25+, and Make. The module pins Go 1.27.3; the standard Go launcher downloads that toolchain when necessary. From the repository root:

```sh
make build-go
gh extension install .
gh arrow --version
gh arrow --help
gh arrow cat github:flarebyte:gh-arrow:README.md
```

This installs a local development extension. The configured build targets are `darwin-arm64`, `linux-amd64`, and `windows-amd64`; native tests must pass before publishing support for them.

## Usage

```sh
gh arrow cp github:owner:project:path/to/file.yaml ./local.yaml
gh arrow cat local:project2:file.md
gh arrow cat github:owner:project:README.md --ref main
gh arrow ls 'github:owner:project:docs/**/*.md'
gh arrow ls --org tools-org --topic cli README.md --json
```

Sources use `github:<owner>:<repo>:<path>` or `local:<alias>:<path>`. Paths use forward slashes; colon is reserved. Quote shell globs. `ls` supports filename `*` and recursive directory `**`; `cat` and `cp` treat filename metacharacters literally. Organization/user discovery checks exact file paths.

GitHub sources prefer configured local working trees. A mapping deliberately selects local files: missing mapped files fail without a remote fallback. `--ref` selects a remote branch, tag, or commit and is rejected for mapped sources and discovery.

Mappings are loaded from `./config.yaml` relative to the working directory, or from `--config <path>`. Roots must be absolute local paths. A missing default config means no mappings; a missing explicitly selected config is an error. Review this trusted configuration before running in an unfamiliar checkout. See the [configuration example](doc/design-meta/examples/config.yaml).

Remote access uses GitHub.com and existing gh credentials. For private repositories, authenticate separately with `gh auth login`. Arrow never prompts, launches a browser, or stores tokens in its config. Public access can work without stored credentials. GitHub Enterprise hosts are outside this version's scope.

`cp` copies one file, requires an existing destination parent, and refuses an existing destination unless `--force` is given. It stages complete bytes before publication and rejects symlink destinations. Source permissions and timestamps are not copied; new files use private permissions.

`cat` writes exact bytes, with no added newline. Successful `cp` writes no stdout. Listings are sorted; text escapes control characters, while `--json` preserves Unicode path values. JSON listings are arrays, including `[]` for no matches; discovery records additionally contain `repository`. Diagnostics go to stderr with stable `ARROW_*` codes. Exit status is 0 for success, 1 for I/O/network/authentication failures, and 2 for invalid arguments or configuration.

## Documentation

- [Contributing](CONTRIBUTING.md): development setup, checks, tests, and specification changes.
- [Technical design](TECHNICAL_DESIGN.md): code boundaries, safety guarantees, and runtime limits.
- [Specification](doc/design/specification.md): command contracts and acceptance scenarios.
- [Documentation index](doc/README.md): design sources, decisions, publishing, and history.

AI agents should start with [AGENTS.md](AGENTS.md). This project includes an [MIT license](LICENSE).
