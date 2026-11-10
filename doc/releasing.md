# Publishing a release

See [Contributing](../CONTRIBUTING.md) for setup and local checks. Changing
documentation or the version file does not publish a release.

Use a normal SemVer version such as `0.1.0` for installation with
`gh extension install flarebyte/gh-arrow`. Prereleases require an explicit
`--pin` to select their tag.

## Prepare the candidate

1. Set the chosen SemVer version in [main.project.yaml](../main.project.yaml).
   `project.go` embeds this file, so `--version` and the release workflow use the
   same version source.
2. Review the code and documentation, run the contributor checks, and require
   successful Linux/macOS native checks and any Windows regressions tied to
   confirmed implementation bugs. Windows artifacts are cross-built without a
   blanket Windows test/security/smoke gate. Perform an opt-in
   check against a real accessible private repository before first publication.
3. Commit and merge the reviewed code and version to the default branch. The
   release workflow refuses other branches.

For a local artifact check, use a clean checkout and run:

```sh
make build-release
```

This builds through the pinned flarebyte tooling and stages only verified,
checksum-listed assets in `build/release`. An existing staging directory is
refused to prevent mixing stale and fresh files. The command does not publish.
Build targets and artifact paths come from [.gh-flarebyte.cue](../.gh-flarebyte.cue).

## Publish the tested commit

Manually dispatch [Manual release](../.github/workflows/release.yml) from the
default branch in GitHub Actions. Native tests and vulnerability scans must pass
before verification and publication proceed.

The verification job validates SemVer, rejects the `0.0.0-dev` sentinel and
existing tags/releases, runs checks, then builds and checksums assets. The
publishing job downloads those same assets, verifies their checksums and the
checkout commit again, and creates the tag/release from the tested commit.
Write permission is confined to the publishing job. SemVer prerelease versions
are marked as GitHub prereleases; hyphens in build metadata do not change that.

After publication, check the uploaded assets/checksums and verify installation
of the published extension with a short help/version/file-operation smoke check.
Do not treat local builds, passing tests, or staged assets as evidence that a
release exists.

## Recover an interrupted publication

If publishing fails after tag creation, preflight will refuse a simple retry.
Follow [Release recovery](release-recovery.md) to inspect the state, verify the
commit, and remove only unpublished state before rerunning. Never replace a
published release under the same version.

Packaging follows GitHub's [extension conventions](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions).
