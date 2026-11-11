# Documentation map

Start with the document matching your task:

| Task | Document |
| --- | --- |
| Install locally and use the commands | [README](../README.md) |
| Set up development, write tests, and run checks | [Contributing](../CONTRIBUTING.md) |
| Navigate implementation boundaries and safety guarantees | [Technical design](../TECHNICAL_DESIGN.md) |
| Find per-file Go summaries and analysis coverage limits | [Thoth analysis](../thoth-meta/README.md) |
| Read command contracts and acceptance scenarios | [Generated specification](design/specification.md) |
| Find and edit specification sources | [Design source guide](design-meta/AGENTS.md) |
| Inspect accepted decisions and their stable IDs | [Decision records](design-meta/examples/decisions.csv) |
| Prepare and publish artifacts | [Publishing a release](releasing.md) |
| Recover a failed publishing run | [Release recovery](release-recovery.md) |
| Understand the original proposal | [Historical specification](history/initial-spec.md) |

The generated specification is assembled from `doc/design-meta`; edit its source
records and run `make doc-design`. The historical proposal is provenance, not
the current contract. [DECISIONS.md](../DECISIONS.md) remains a compatibility
entry point to the active decision records. AI agents should first read the
repository [AGENTS.md](../AGENTS.md).
