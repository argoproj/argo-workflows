# Release Instructions

## Cherry-Picking Fixes

✋ Before you start, make sure you have created a release branch (e.g. `release-3.3`) and it's passing CI.
Please make sure that all patch releases (e.g. `v3.3.5`) should be released from their associated minor release branches (e.g. `release-3.3`)
to work well with our versioned website.

Then get a list of commits you may want to cherry-pick:

```bash
./hack/cherry-pick.sh release-3.3 "fix"
./hack/cherry-pick.sh release-3.3 "chore(deps)"
./hack/cherry-pick.sh release-3.3 "build"
./hack/cherry-pick.sh release-3.3 "ci"
```

To automatically cherry-pick, run the following:

```bash
./hack/cherry-pick.sh release-3.3 "fix" false
```

Then look for "failed to cherry-pick" in the log to find commits that fail to be cherry-picked and decide if a
manual patch is necessary.

Ignore:

* Fixes for features only on `main`.
* Dependency upgrades, unless they fix known security issues.
* Build or CI improvements, unless the release pipeline is blocked without them.

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

### Feature Releases

For feature releases (e.g., v3.6.0, v3.7.0) and not patch releases (e.g., v3.6.1, v3.6.5), you need to update the feature descriptions with the new version.

For release candidates, use:

```bash
make features-update VERSION=v3.6.0
git add docs/new-features.md
git commit -m "chore: Update feature descriptions for v3.6.0"
git push
```

This will update all pending feature descriptions with the current version and include them in the upcoming release notes.
The features will remain in the pending directory, allowing for further updates if needed.

For the final release, use:

```bash
make features-release VERSION=v3.6.0
git add .features
git add docs/new-features.md
git commit -m "chore: Release feature descriptions for v3.6.0"
git push
```

This will update the feature descriptions and move them from the pending directory to the released directory for the specific version.
This is the final step that should be done when releasing a new version.

### Release Build

GitHub Actions will automatically build and publish your release. This takes about 1h. Set your self a reminder to check
this was successful.

## Update Changelog

Once the tag is published, GitHub Actions will automatically open a PR to update the changelog. Once the PR is ready,
you can approve it, enable auto-merge, and then run the following to force trigger the CI build:

```bash
git branch -D create-pull-request/changelog
git fetch upstream
git checkout --track upstream/create-pull-request/changelog
git commit -s --allow-empty -m "chore: Force trigger CI"
git push upstream create-pull-request/changelog
```

## Announce on Slack

Once the changelog updates have been merged, you should announce on our Slack channels, [`#argo-workflows`](https://cloud-native.slack.com/archives/C01QW9QSSSK) and [`#argo-announcements`](https://cloud-native.slack.com/archives/C02165G1L48).
See [previous](https://cloud-native.slack.com/archives/C02165G1L48/p1701112932434469) [announcements](https://cloud-native.slack.com/archives/C01QW9QSSSK/p1701112957127489) as examples of what to write in the patch announcement.
