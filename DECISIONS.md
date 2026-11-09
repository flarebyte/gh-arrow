# Design specification

The active specification has moved to [doc/design-meta/app.cue](doc/design-meta/app.cue) and its structured inputs. Read the [generated specification](doc/design/specification.md) for the complete design and decision table.

The target is now a Go GitHub CLI extension named `gh-arrow`, invoked as `gh arrow`. Remote access reuses GitHub CLI credentials, superseding the original public-only and no-authentication constraints. The hosted repository is now `gh-arrow`; release publication remains a future task.

The complete initial proposal is preserved in [doc/history/initial-spec.md](doc/history/initial-spec.md) for provenance. It is historical, including its `gha` commands, raw URL and web listing fallbacks, and authentication exclusions. Resolve conflicts using the active specification.

Edit the source tables and regenerate with `sh scripts/doc-design.sh`; do not maintain a second specification here.
