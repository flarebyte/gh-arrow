# Implementation guide

The [specification](doc/design/specification.md) defines command behavior. Tests
reference its acceptance IDs (A01–A19); changes should begin with a failing test.
The commands are implemented, but no release has been published.

## Boundaries

| Package | Responsibility |
| --- | --- |
| `cmd/gh-arrow` | Signals, process streams, and final exit status |
| `internal/cli` | Argument validation, orchestration, and output rendering |
| `internal/source` | Explicit source identities and repository-relative paths |
| `internal/config` | Strict, read-only YAML mappings |
| `internal/local` | Descriptor-relative reads, filesystem traversal, staged copies |
| `internal/github` | gh credentials, HTTP errors, pinned Git trees, discovery |
| `internal/listing` | Shared glob matching, sorting, and text escaping |
| `internal/diagnostic` | Stable error codes and exit status mapping |

The root `project.go` embeds `main.project.yaml` as build metadata; it does not
define a supported library API. `--version` reports that version and available Go
VCS build settings without runtime files or network access.

The shared backend interface lives at its consumer in `internal/cli`. Both
backends expose open, stat, and list operations. Discovery belongs to the GitHub
backend and deliberately bypasses local mappings. Avoid adding a service layer
or interface for every concrete type.

## Safety and consistency

Validate arguments before reading config or credentials; help needs neither.
Mappings select local working trees permanently for that operation, including
when a file is missing. GitHub identity matching in mappings is case-insensitive.

Local reads walk from a pinned directory descriptor. Unix opens use `openat`
with no-follow/nonblocking flags; Windows uses root-relative NT opens with
`OBJ_DONT_REPARSE`. Regular-file checks happen before reading. Listing metadata
uses `os.Root` so concurrent symlink replacement cannot escape the mapped root.
The configured root itself is trusted and may be a filesystem-resolved alias.

Copies use a private temporary file in the destination directory, flush and close
it, and publish it with an atomic filesystem operation. Default no-clobber is
race-safe. Unix uses mode 0600; Windows applies a creator-only protected ACL.
Failed transfers preserve existing destinations. Destination permissions and
timestamps are not copied from the source. These are atomic replacement
guarantees, not a guarantee against power loss or hostile filesystem mounts.

Remote operations obtain credentials from gh once and send them only to fixed
GitHub.com HTTPS endpoints. Redirects are refused; HTTP response bodies and
credentials are never included in diagnostics. Public reads can run without a
stored token. Authentication errors guide the user to run `gh auth login`
separately. Resolve the selected branch/tag/ref to one commit tree using the metadata-only
commit-list endpoint before reads; the single-commit endpoint can include file
diffs and is deliberately avoided.
Inspect Git tree entry modes to exclude symlinks and submodules; fetch file bytes
through the raw Git blob API rather than the Contents API's symlink behavior.

Directory and discovery results are collected and sorted before stdout is
written. Enumeration failures therefore produce no partial success listing.
`cat` streams exact bytes and may contain partial output on a failed transfer;
it still exits nonzero. Text listings escape control characters; JSON retains
the corresponding Unicode paths. Local non-UTF-8 filenames fail enumeration
rather than being silently rewritten in JSON.

## Bounds and cancellation

- Config: 1 MiB; each decoded API metadata response: 64 MiB.
- API calls: 30-second timeout, at most two retries for network/5xx failures,
  with 100/200 ms backoff. Rate limits fail immediately; no indefinite sleeps.
- Credential lookup: 10 seconds. SIGINT/SIGTERM cancel work and staged copies.
- Discovery: four workers, at most 1,000 pages per endpoint and 100,000 distinct
  owned repositories. Public and authenticated user enumeration are combined.
- Traversal: at most 256 directory levels and one million visited entries;
  remote repositories additionally cap cached tree metadata at one million
  entries, a 64 MiB accounting budget for strings and per-entry overhead, and
  10,000 distinct tree requests. Limit failures are explicit.
- Files are streamed rather than buffered in memory. GitHub API file-size limits
  are reported as failures; no successful truncation or content cache is allowed.

## Verification

Run `make review`, `make test-race`, `make coverage`, and `make build` locally.
Source/glob fuzz targets use Go's native fuzzing. `make sec` downloads the pinned
checker and advisory data; CI scans each native platform, since a host-only scan
cannot assess Windows-only calls. Native CI also runs the unit tests and Python
executable smoke checks on Linux, macOS, and Windows.

Unit tests replace HTTP transport and credential lookup with fixtures. They
cover binary bytes, deterministic output, error classification, pagination,
private discovery, symlinks, FIFOs, concurrent replacement, and failed transfers.
No normal test accesses GitHub or the user's real credentials. Before publication,
separately verify a real accessible private repository with explicit authorization.

Enterprise hosts remain outside v1.
