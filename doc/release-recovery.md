# Recover an interrupted release

Publishing creates a tag before creating the GitHub release and uploading assets.
If a later request fails, an orphan tag or incomplete draft can remain. Preflight
deliberately refuses existing tags and releases, so rerunning alone will fail.

Use this procedure only for a failed **Manual release** run. It deletes an
unpublished draft and tag; never use it to replace a published release.

1. Open the failed run and record its full commit SHA and release tag. Confirm
   that the run failed during publishing, rather than assuming an existing tag
   belongs to it. Set these values in your shell:

   ```sh
   RELEASE_REPO=flarebyte/gh-arrow
   RELEASE_TAG=v0.1.0 # Replace with the failed run's tag.
   RELEASE_COMMIT=FULL_SHA_FROM_FAILED_RUN
   ```

2. Verify that the tag is the lightweight commit tag created by this workflow
   and points to that exact commit. Stop if either check fails. Do not delete
   an annotated tag or a tag pointing to another commit.

   ```sh
   gh api "repos/$RELEASE_REPO/git/ref/tags/$RELEASE_TAG" --jq '.object'
   test "$(gh api "repos/$RELEASE_REPO/git/ref/tags/$RELEASE_TAG" --jq '.object.type')" = commit
   test "$(gh api "repos/$RELEASE_REPO/git/ref/tags/$RELEASE_TAG" --jq '.object.sha')" = "$RELEASE_COMMIT"
   ```

3. Inspect the release, including its publication status:

   ```sh
   gh release view "$RELEASE_TAG" --repo "$RELEASE_REPO" --json url,isDraft,publishedAt
   ```

   If it is published (`isDraft: false`), stop: inspect the published assets and
   use a new version for any correction. Do not delete or overwrite that release.
   If the command fails, distinguish an actual missing release from an
   authentication, permission, rate-limit, or network error before continuing.

4. If a draft exists, check its assets in GitHub and delete **only that draft**.
   This guard must succeed before running the deletion command:

   ```sh
   test "$(gh release view "$RELEASE_TAG" --repo "$RELEASE_REPO" --json isDraft --jq '.isDraft')" = true &&
     gh release delete "$RELEASE_TAG" --repo "$RELEASE_REPO" --yes
   ```

   Skip draft deletion if the release is confirmed absent. Confirm the release
   is now absent, then recheck the tag's type and SHA from step 2 immediately
   before deleting the orphan tag:

   ```sh
   test "$(gh api "repos/$RELEASE_REPO/git/ref/tags/$RELEASE_TAG" --jq '.object.type')" = commit &&
     test "$(gh api "repos/$RELEASE_REPO/git/ref/tags/$RELEASE_TAG" --jq '.object.sha')" = "$RELEASE_COMMIT" &&
     gh api --method DELETE "repos/$RELEASE_REPO/git/refs/tags/$RELEASE_TAG"
   ```

   Coordinate with other maintainers so no one publishes or changes the tag
   during recovery. If repository rules prevent deletion, involve a repository
   administrator; do not bypass the rules.

5. Rerun **all jobs** of the original failed workflow run, which preserves its
   original commit. If its artifacts have expired, rebuilding all jobs generates
   fresh verified artifacts. A new manual dispatch uses the selected branch's
   current commit, so review any intervening changes before choosing that route.

If no tag was created, confirm there is no release for that version and rerun
all jobs without deleting anything. The `0.0.0-dev` scaffold version remains
blocked regardless of recovery.
