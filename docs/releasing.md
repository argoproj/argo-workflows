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

GitHub Actions will automatically build and publish your release. This takes about 1h. Set your self a reminder to check
this was successful.

## Update Changelog

Once the tag is published, GitHub Actions will automatically open a PR to update the changelog. Once the PR is ready,
you can approve it, enable auto-merge, and comment `/test` to run CI on it.

## Announce on Slack

Once the changelog updates have been merged, you should announce on our Slack channels, [`#argo-workflows`](https://cloud-native.slack.com/archives/C01QW9QSSSK) and [`#argo-announcements`](https://cloud-native.slack.com/archives/C02165G1L48).
See [previous](https://cloud-native.slack.com/archives/C02165G1L48/p1701112932434469) [announcements](https://cloud-native.slack.com/archives/C01QW9QSSSK/p1701112957127489) as examples of what to write in the patch announcement.
