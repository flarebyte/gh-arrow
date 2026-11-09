# gh-arrow specification

Generated from doc/design-meta; edit the source and regenerate.

## Interface

### Interface details

#### Command surface

| behavior | concept | id | syntax |
| --- | --- | --- | --- |
| Copy one file; refuse existing destination unless --force. | files | C01 | gh arrow cp <source> <destination> [--force] |
| Write exact file bytes to stdout. | files | C02 | gh arrow cat <source> |
| List matching files; quote shell globs. | files.list | C03 | gh arrow ls <source> [--json] |
| Find an exact repository-relative file across an organization or user. | discovery | C04 | gh arrow ls --org <owner> [--topic <topic>] <path> [--json] |
| Show syntax and copyable examples without network access. | automation | C05 | gh arrow <command> --help |
| Override the default ./config.yaml mapping file. | config | C06 | gh arrow <command> --config <path> |
| Select a remote branch tag or commit; reject for mapped sources and organization discovery in v1. | source | C07 | gh arrow <command> --ref <ref> |
| Report the embedded project version and available commit metadata without config or network access. | extension | C08 | gh arrow --version |

#### Mapping configuration

```yaml
# Roots must be absolute on the local OS; use C:/work/project on Windows.
github:
  mappings:
    - repo: owner:project
      root: /absolute/path/to/project
local:
  mappings:
    - repo: project2
      root: /absolute/path/to/project2
```

#### JSON listing contract

```ts
// Specification examples only; this project is implemented in Go.
// --json always emits an array; mode determines the record shape.
export type SourceListing = { path: string }; // relative to matched directory prefix
export type DiscoveryListing = { repository: string; path: string }; // repo-root-relative
export const sourceExample: SourceListing[] = [{ path: "guide.md" }];
export const discoveryExample: DiscoveryListing[] = [
  { repository: "tools-org/example", path: "README.md" },
];
```

#### Usage examples

| concept | example | expected | id |
| --- | --- | --- | --- |
| files | gh arrow cp github:owner:project:path/to/file.yaml ./local.yaml | Copy from GitHub or the configured local working tree. | U01 |
| files | gh arrow cat local:project2:file.md | Display a file from a named local mapping. | U02 |
| files | gh arrow cp local:project2:file.md ./file.md | Copy from a private or non-GitHub local project. | U03 |
| files | gh arrow cat github:owner:project:file.yaml | Display remote bytes without saving a file. | U04 |
| files.list | gh arrow ls github:owner:project:docs/ | List files directly within docs. | U05 |
| files.list | gh arrow ls 'local:project2:docs/*.md' | List Markdown files directly within docs. | U06 |
| files.list | gh arrow ls 'github:owner:project:src/**' | List files recursively below src. | U07 |
| discovery | gh arrow ls --org tools-org package.json | Find root package.json files across repositories. | U08 |
| discovery | gh arrow ls --org tools-org --topic cli README.md --json | Return machine-readable matches for repositories with topic cli. | U09 |

## Navigation and follow-up

### Navigation and follow-up details

#### Concept registry

| description | id | parent |
| --- | --- | --- |
| GitHub CLI packaging and invocation | extension |  |
| Source grammar and resolution | source |  |
| Read-only local mappings | config |  |
| File operations | files |  |
| Directory listing and glob patterns | files.list | files |
| Organization or user repository discovery | discovery |  |
| Human and agent interaction contracts | automation |  |

#### Open questions

| concept | id | question | status |
| --- | --- | --- | --- |
| extension | Q01 | Which Go and gh minimum versions and release OS/architecture targets should be supported? | Initial implementation: Go 1.25 language minimum with Go 1.27.3 toolchain; gh 2.103.3 tested baseline; linux-amd64 and darwin-arm64 have full native CI. windows-amd64 remains cross-built; Windows checks are limited to regressions for confirmed implementation bugs. |
| discovery | Q02 | Which bounded concurrency and retry limits should discovery use? | Resolved: four workers; 30-second requests; two retries for network/5xx failures; no rate-limit retry. Finite metadata and traversal limits are documented in TECHNICAL_DESIGN.md. |
| automation | Q03 | Should future versions support GitHub Enterprise hosts and richer structured error output? | Deferred beyond v1. |

## Overview

### Overview details

#### Design decisions

| concept | decision | id | rationale | status | supersedes |
| --- | --- | --- | --- | --- | --- |
| extension | Build a Go GitHub CLI extension named gh-arrow; invoke as gh arrow. | D01 | Use the gh extension lifecycle and a familiar command entry point. | accepted | Independent gha binary and arrow-gha product name. |
| extension | Use the GitHub API with existing gh credentials obtained through gh auth token; never manage tokens in arrow config. | D02 | Reuse GitHub CLI authentication for public and accessible private repositories. | accepted | No API authentication; public-only remote access; raw URL and web-page fallback. |
| automation | Treat doc/design-meta as the specification source and generate doc/design/specification.md with flyb. | D03 | Support structured retrieval for agents and readable review for humans. | accepted | DECISIONS.md as the active specification. |
| source | Prefer explicit local mappings; never fall back remotely after a mapped file is missing. | D04 | A local mapping deliberately selects the working tree. | accepted | Clarifies the original missing-local-file edge case. |
| extension | Use the current hosted repository name gh-arrow for the extension and executable. | D05 | The repository has been renamed to gh-arrow; align invocation with GitHub extension naming conventions. | accepted | Earlier plan to rename the hosted repository before publication. |

#### Overview

gh-arrow is a Go extension invoked as `gh arrow`. It copies, displays and lists files from GitHub or mapped local working trees, and discovers files across organizations or users. File commands and discovery are implemented with offline acceptance tests and native CI verification; no release has been published. The original proposal is preserved in `doc/history/initial-spec.md`; decisions below explicitly replace its standalone name and public-only remote authentication restrictions.

#### GitHub extension references

Packaging follows [Creating GitHub CLI extensions](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions) and the [extension command manual](https://cli.github.com/manual/gh_extension). The repository target name is `gh-arrow`, the executable is `gh-arrow`, and the invocation is `gh arrow`. The hosted repository is now `gh-arrow`; release publication remains a future implementation task.

## Requirements

### Requirements details

#### Acceptance scenarios

| concept | expected | id | scenario |
| --- | --- | --- | --- |
| source | Read local bytes without a network call. | A01 | Mapped GitHub repository contains the requested file |
| source | ARROW_NOT_FOUND and exit 1; no remote fallback. | A02 | Mapped GitHub repository lacks the requested file |
| config | No mappings; exit 2; no mappings; exit 2 respectively. | A03 | Missing default config; missing explicit config; empty config; malformed config |
| source | ARROW_USAGE and exit 2. | A04 | Invalid source or unsupported ftp scheme |
| files | Exit 1 with ARROW_EXISTS or ARROW_PERMISSION; preserve existing file. | A05 | Destination exists without --force or parent is unwritable |
| automation | Exit 1; diagnostic code on stderr; no partial stdout. | A06 | Network failure or rate limiting while enumerating |
| files.list | Single file; zero results; zero results; exit 2 respectively. | A07 | File operand; empty directory; unmatched glob; invalid glob |
| files.list | Complete traversal or ARROW_INCOMPLETE with exit 1. | A08 | Recursive remote tree is truncated |
| discovery | Exit 1; empty success; empty success; empty success respectively. | A09 | Invalid owner; valid owner without repositories; no topic matches; no file matches |
| discovery | Do not report a file match. | A10 | Discovery operand refers to a directory |
| extension | Read using existing gh credentials; never request arrow-specific credentials. | A11 | Authenticated accessible private repository |
| source | Reject access; never read outside the root. | A12 | Source traversal or symlink escaping mapped root |
| automation | Identical sorted text and JSON outputs. | A13 | Repeated listing over unchanged fixtures |
| files | Exit 1; preserve existing bytes and remove temporary files. | A14 | Failed transfer during cp --force; destination is a symlink; source and destination are the same file |
| source | Reject without blocking or reading outside the mapped root. | A15 | Local source is a FIFO or a symlink component; a path is replaced by an escaping symlink during open |
| source | Reject cat/cp sources; omit listing and discovery matches. | A16 | Remote path is a symlink or submodule |
| discovery | Include the match through authenticated repository enumeration. | A17 | A user owns an accessible private repository containing the requested file |
| automation | Escape controls in text; preserve the path value in valid JSON. | A18 | Listing path contains a newline or terminal escape character |
| extension | Exit 0 with embedded project version; do not read runtime config or credentials. | A19 | Version requested with an absent explicit config and no network access |

#### Scope limits

| concept | id | requirement |
| --- | --- | --- |
| files | N01 | No cloning, recursive directory copy, file content cache, or configuration writes. |
| files | N02 | No archive inspection, binary introspection, full-text search, metadata filtering, or special case-insensitive filesystem handling. Binary files remain copyable and displayable as bytes. |
| extension | N03 | No custom credential store or authentication flow; use gh credentials. No GitHub Enterprise host support in v1. |
| discovery | N04 | Discovery does not support recursive globs, license or star filters, or fork deduplication. |
| discovery | N05 | Remote results reflect API observations and are not guaranteed real-time snapshots across repositories. |
| automation | N06 | Never overwrite an existing destination without --force. No interactive overwrite confirmation. |

#### Behavioral rules

| concept | id | requirement |
| --- | --- | --- |
| source | R01 | Accept github:<owner>:<repo>:<path> and local:<alias>:<path>; require nonempty identity segments and valid GitHub owner/repository names. Paths use forward slashes; colon is reserved. Empty path is valid only for ls and means repository root. |
| source | R02 | Resolve github owner:repo through github.mappings first; otherwise use the GitHub API on github.com. Resolve local alias only through local.mappings. Do not infer identities from URLs. |
| source | R03 | Remote operations use the repository default branch unless --ref is supplied. Resolve the selected ref to one commit before reading paths; use that commit throughout the operation. |
| source | R04 | Reject absolute source paths and parent traversal. Local opens must enforce containment within the mapped root even if paths change concurrently. Reject symlink path components and nonregular file sources; skip symlinks and special files in listings. Remote reads and discovery must also exclude symlinks and submodules using Git tree entry types. |
| config | R05 | Read ./config.yaml relative to the working directory unless --config overrides it. Missing default config means no mappings; missing explicit config is an error. Empty config means no mappings. |
| config | R06 | Reject malformed YAML, duplicate YAML keys, unknown keys, duplicate repo mappings, empty identities and nonabsolute roots. Compare GitHub mapping identities case-insensitively. Never write configuration at runtime. |
| files | R07 | cp requires an existing destination parent; do not create directories. Reject directory or special-file destinations, destination symlinks and copying a file onto itself. Stage complete bytes in a private temporary file beside the destination; publish only after successful transfer. Refusal without --force must be race-safe. Failed copies preserve any existing destination and remove temporary files. |
| automation | R08 | cat emits raw bytes only. cp emits no stdout on success. Send diagnostics to stderr. |
| files.list | R09 | ls outputs files only in lexicographic order with forward slashes and no headers or decoration. Paths are relative to the literal directory prefix before a glob; for a file operand return its basename. |
| files.list | R10 | Support * within a filename and ** as a complete recursive directory segment; ** includes zero directories. Reject ?, bracket patterns and embedded **. Directory-segment wildcards other than ** are unsupported. |
| files.list | R11 | A directory operand lists immediate files; a file operand lists that file. Empty directories and unmatched valid globs succeed with zero results; missing literal paths fail. Apply the same semantics locally and remotely. |
| files.list | R12 | Use API directory/tree enumeration rather than scraping GitHub pages. Detect truncated responses; complete enumeration or return an explicit incomplete-results error. Skip submodules and symlinks. |
| discovery | R13 | --org accepts an organization or user; enumerate all accessible repositories with pagination. Include archived repositories and forks; do not deduplicate forks. |
| discovery | R14 | Filter by exact topic when provided and check an exact nonempty file path relative to each repository root on its default branch. Reject globs. A directory at that path is a nonmatch for that repository. Do not fetch file contents. |
| discovery | R15 | Organization discovery always checks remote repositories and ignores local mappings. Include accessible private repositories through gh credentials. For user owners combine public owner enumeration with authenticated accessible repository enumeration filtered to that owner; the public user endpoint alone is insufficient. Visibility remains limited by the active credentials. |
| discovery | R16 | For discovery output owner/repo:path one per line sorted by owner/repo then path. No matches succeeds with empty output. |
| automation | R17 | Never prompt or launch a browser during file operations. If authentication is needed, fail with guidance to run gh auth login separately. Do not expose credentials in diagnostics. |
| automation | R18 | ls --json returns a JSON array of records as defined in listing.ts; empty result is []. Reject --json for cat and cp to protect byte-stream semantics. |
| automation | R19 | Exit 0 for success including no matches; 1 for I/O, network, authentication, rate-limit or incomplete enumeration failures; 2 for invalid arguments, sources or configuration. |
| automation | R20 | On enumeration failure return nonzero with no success listing on stdout; diagnostics identify the failing operation. Report rate limits without silently skipping repositories. |
| automation | R21 | Use human-readable stderr messages prefixed by stable error codes: ARROW_USAGE, ARROW_CONFIG, ARROW_NOT_FOUND, ARROW_EXISTS, ARROW_PERMISSION, ARROW_AUTH, ARROW_NETWORK, ARROW_RATE_LIMIT, ARROW_INCOMPLETE, ARROW_IO. |
| extension | R22 | Build a gh-arrow executable and publish precompiled release assets following GitHub CLI platform naming conventions. Document local build and gh extension install . once implementation exists. |
| source | R23 | Use GitHub.com explicitly for remote API access through gh credentials. Encode path and ref values as data; never interpolate them into shell commands. Never forward credentials to unrelated redirect hosts. |
| automation | R24 | Escape control characters in text listings and diagnostics so each result occupies one line. JSON preserves path values using JSON escaping. cat remains an unmodified byte stream; a failed stream may contain partial bytes and must exit nonzero. |
| automation | R25 | Use bounded request timeouts, retries and concurrency; support cancellation. API size or enumeration limits must produce a clear nonzero error unless a supported fallback completes the operation. Never silently truncate file bytes or listing results. |

