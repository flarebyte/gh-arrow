# Working on gh-arrow

This repository contains specifications and a modular Go implementation. The target is a Go GitHub CLI extension: repository/executable `gh-arrow`, invocation `gh arrow`. File operations and discovery are implemented; do not assume a published release exists.

- Start with README.md, then doc/design-meta/AGENTS.md for focused specification retrieval.
- Edit doc/design-meta inputs; regenerate doc/design/specification.md with `sh scripts/doc-design.sh`.
- Keep stable rule and decision IDs so implementation and tests can reference them.
- Use the acceptance scenarios when implementing behavior. Preserve raw `cat` bytes, deterministic listings, noninteractive operation, explicit local resolution, and gh-managed credentials.
- doc/history/initial-spec.md is historical; its standalone name and authentication exclusions are superseded.
- The hosted repository is gh-arrow. Do not claim a release has happened based on documentation changes.

- Go entry point: cmd/gh-arrow/main.go. Use gh flarebyte config validate, format, lint, test, and build for local checks. Keep generated binaries and caches out of commits.

- Add regression/acceptance tests before changing behavior. Prefer small packages and interfaces at consumer boundaries. Use make review, make test-race, and make build; make sec also requires advisory database access. Normal tests use fixtures and must never depend on real GitHub credentials or network access.
