# Working on gh-arrow

This repository currently contains specifications only. The target is a Go GitHub CLI extension: repository/executable `gh-arrow`, invocation `gh arrow`. Do not assume the extension is implemented or published.

- Start with README.md, then doc/design-meta/AGENTS.md for focused specification retrieval.
- Edit doc/design-meta inputs; regenerate doc/design/specification.md with `sh scripts/doc-design.sh`.
- Keep stable rule and decision IDs so implementation and tests can reference them.
- Use the acceptance scenarios when implementing behavior. Preserve raw `cat` bytes, deterministic listings, noninteractive operation, explicit local resolution, and gh-managed credentials.
- doc/history/initial-spec.md is historical; its standalone name and authentication exclusions are superseded.
- The hosted repository is gh-arrow. Do not claim a release has happened based on documentation changes.
