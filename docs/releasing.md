# Release Instructions

## Cherry-Picking Fixes

✋ Before you start, make sure the release branch is passing CI.

Get a list of commits you may want to cherry-pick:

```bash
./hack/what-to-cherry-pick.sh release-3.3
```

Ignore:

* Fixes for features only on master.
* Dependency upgrades, unless it fixes a known security issue.

Cherry-pick the first commit. Run `make test` locally before pushing. If the build timeouts the build caches may have
gone, try re-running.

Don't cherry-pick another commit until the CI passes. It is harder to find the cause of a new failed build if the last
build failed too.

Cherry-picking commits one-by-one and then waiting for the CI will take a long time. Instead, cherry-pick each commit then
run `make test` locally before pushing.

## Publish Release

✋ Before you start, make sure the branch is passing CI.

Push a new tag to the release branch. E.g.:

```bash
git tag v3.3.4
git push upstream v3.3.4 # or origin if you do not use upstream
```

GitHub Actions will automatically build and publish your release. This takes about 1h. Set your self a reminder to check
this was successful.

## Update Changelog

Once the tag is published, GitHub Actions will automatically open a PR to update the changelog. Once the PR is ready,
you can approve it, enable auto-merge, and then run the following to force trigger the CI build:

```bash
git checkout --track upstream/create-pull-request/changelog
git commit -s --allow-empty -m "docs: Force trigger CI"
git push upstream create-pull-request/changelog
```
