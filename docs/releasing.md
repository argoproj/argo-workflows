# Release Instructions

## Cherry-Picking Fixes

✋ Before you start, make sure the branch is already passing in CI.

Get a list of fix you might wish to cherry-pick:

```bash
./hack/what-to-cherry-pick.sh release-3.3
```

Ignore:

* Anything that is a fix for something on master, but not on the release branch.
* Dependency upgrades, unless it fixes a know security issue.

Cherry-pick a the first commit. Run `make test` locally before pushing. If the build timeouts, try re-running.

Don't cherry-pick a second commit until the CI passes. It is much harder to know the cause if you do many issues at once.

Cherry-picking and then waiting for CI will be slow.

Run `make test` locally before pushing each cherry-picked commit.

### Publish Release

Push a new tag to the release branch. Github Actions will automatically build and publish your release. This takes about
1h. Set your self a reminder to check this was successful.
