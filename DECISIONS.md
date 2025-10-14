# Architecture decision records

An [architecture
decision](https://cloud.google.com/architecture/architecture-decision-records)
is a software design choice that evaluates:

- a functional requirement (features).
- a non-functional requirement (technologies, methodologies, libraries).

The purpose is to understand the reasons behind the current architecture, so
they can be carried-on or re-visited in the future.

## Initial idea

**Specification: GitHub Utility CLI `gha` for File Copying and Display**

---

**Problem Description**

Create a CLI utility `gha` in Go that allows copying or displaying files from either GitHub repositories or local development directories, supporting both public and private repositories via configuration. The tool must determine the source path based on a configuration file and resolve mappings accordingly before accessing remote or local file systems.

---

**Use Cases**

1. Copy a public GitHub file to a local path using GitHub reference:

   * `gha cp github:owner:project:path/to/file.yaml ./local.yaml`
2. Copy a file from a locally cloned GitHub project using defined mappings:

   * If `config.yaml` maps `owner:project` to a local path, copy from there instead of GitHub.
3. Display contents of a file from local mapping without writing to disk:

   * `gha cat local:project2:file.md`
4. Copy a file from a custom local mapping for private or non-GitHub projects:

   * `gha cp local:project2:file.md ./file.md`
5. Display contents of a public GitHub file without saving:

   * `gha cat github:owner:project:file.yaml`

---

**Edge Cases**

* Mapping for GitHub `owner:project` exists but file does not exist locally.
* Configuration file is missing, empty, or has malformed mappings.
* File path provided does not exist in either remote or mapped local root.
* Invalid source format (e.g., missing segments like `github:owner:project:file`).
* Destination file already exists or is not writable.
* Network error while accessing public GitHub URL.
* Use of unsupported source types (e.g., `ftp:`).

---

**Configuration (YAML)**
A single configuration file `config.yaml` defines mappings:

```
github:
  mappings:
    - repo: owner:project
      root: /Users/path/Documents/github/project

local:
  mappings:
    - repo: project2
      root: /Users/path/Documents/github/project2
```

---

**Constraints / Not to Handle**

* Do not support full repository cloning.
* Do not support authentication for private GitHub repositories.
* Do not support recursive directory copy.
* Do not infer repo names from URLs or vice versa.
* Do not create missing parent directories for destination paths.
* Do not cache GitHub file contents.
* Do not modify or write to the configuration file at runtime.

---

**Expected Behavior Summary**

* If a `github:` source has a local mapping, use the local file path.
* If no local mapping exists for a GitHub repo, use the raw GitHub URL to fetch.
* `local:` sources must be mapped and exist locally.
* `cp` command writes the file to a destination.
* `cat` command outputs file content to stdout without saving.

