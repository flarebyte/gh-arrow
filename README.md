# gh-arrow

A planned GitHub CLI extension to copy, display, and discover files in GitHub repositories and mapped local projects. The command will be `gh arrow`; the extension repository and executable will be named `gh-arrow`.

**Status:** specification only. No executable or installable release exists yet. The hosted repository is currently `arrow-gha`; renaming it to `gh-arrow` is planned before publication.

Planned usage:

```sh
gh arrow cp github:owner:project:path/to/file.yaml ./local.yaml
gh arrow cat local:project2:file.md
gh arrow ls 'github:owner:project:docs/**/*.md'
gh arrow ls --org tools-org --topic cli README.md --json
```

GitHub sources prefer configured local working trees. Other remote operations use GitHub CLI API access and its existing credentials, including accessible private repositories. Operations never prompt. Listings support sorted text or JSON; `cat` preserves raw bytes. See the [configuration example](doc/design-meta/examples/config.yaml).

Read the [generated specification](doc/design/specification.md) for command contracts, requirements, acceptance scenarios, and open questions. The editable source lives in [doc/design-meta](doc/design-meta/AGENTS.md), assembled by [app.cue](doc/design-meta/app.cue). The [original proposal](doc/history/initial-spec.md) is historical.

To update the design, edit its sources and regenerate from the repository root:

```sh
# Requires Python 3 and flyb on PATH.
sh scripts/doc-design.sh
```

Commit both the sources and generated Markdown. AI agents should start with [AGENTS.md](AGENTS.md) and retrieve only the relevant concept records.

The extension will follow GitHub's [extension packaging conventions](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions). Installation and local build instructions will be added when an executable exists.
