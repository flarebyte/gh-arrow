# Repository analysis

The existing pipelines were reviewed before generation and left unchanged.

| Pipeline | Relevance | Generated coverage |
| --- | --- | --- |
| `pipeline-go-maat.thoth.cue` | Relevant | 15 production Go files, including the entry point, embedded version metadata, and internal packages |
| `pipeline-go-test-maat.thoth.cue` | Relevant | 12 Go test files, including CLI acceptance tests and platform-specific regressions |
| `pipeline-ts-e2e-maat.thoth.cue` | No matching inputs | No `e2e/*.test.ts` or `e2e/*.suite.ts` files exist; pipeline skipped |

Executable smoke coverage exists in `scripts/smoke.py`; release-tooling tests are
also Python. The TypeScript e2e pipeline does not cover these files. The
TypeScript design example describes data shapes rather than executable tests.

## Reading the metadata

`go/` and `go-test/` contain per-file imports, function names, and curated purpose
sentences. Start with the relevant purpose, then inspect the source for details.
CLI orchestration consumes the backend interface; local filesystem safety and
remote GitHub access remain in their respective packages.

Both pipelines select `import_files_list,function_map`. They do not request
`method_map` or `testcase_titles_list`, so these summaries are not complete
method inventories or test-case catalogs. Static analysis includes both Unix
and Windows files; generation does not execute tests or prove platform support.
Python scripts and embedded CUE schemas are outside these pipelines' coverage.

## Refresh

Run from the repository root with `thoth`, Node.js, and npm available:

```sh
thoth run --config pipeline-go-maat.thoth.cue
thoth run --config pipeline-go-test-maat.thoth.cue
```

The configs invoke `npx maat-ostraca`, which may require registry access and is
not version-pinned. Debug envelopes are written to `temp/`; durable summaries
are the YAML files here. Inspect debug records for shell errors because the
configs keep going on errors and substitute empty lists for missing rule output.
Preserve curated purpose fields on refresh.

Initial generation checked all 27 records for successful analyzer exit status,
empty stderr, and Go language output. No source behavior was changed.
