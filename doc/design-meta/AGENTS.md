# Finding and updating the specification

Run commands from the repository root. `app.cue` is the input catalog and Markdown assembly; `examples/concepts.csv` is the canonical concept registry. Generated output is `doc/design/specification.md`.

Find a concept, then retrieve related records:

```sh
rg -n -i 'listing|authentication|agent' doc/design-meta/examples/concepts.csv
rg -l -F 'files.list' doc/design-meta/examples -g '*.csv'
rg -n '(^|,)"?files\.list"?(,|$)' doc/design-meta/examples -g '*.csv'
rg -n '(^|,)"?files(\.[a-z0-9-]+)*"?(,|$)' doc/design-meta/examples -g '*.csv'
head -n 1 doc/design-meta/examples/rules.csv
```

Text search identifies candidates; inspect headers and use Python's csv module for exact column filtering. Follow related requirements across concepts. The config.yaml and listing.ts examples carry additional contracts; inspect their notes in app.cue. TypeScript describes JSON shapes, not an implementation language.

Tables cover decisions, commands, use cases, rules, scope limits, acceptance scenarios, and open questions. Keep stable record IDs and one physical line per CSV record. Reuse concepts rather than inventing synonymous labels. Small inputs stay flat until domain folders help navigation.

After changes run:

```sh
sh scripts/doc-design.sh
```

This validates CSV structure, identifiers, and concept references before flyb validation and Markdown generation. Review and commit generated output with source changes. Do not edit generated Markdown directly. Preserve historical provenance in doc/history/initial-spec.md; record intentional departures in decisions.csv.
