# Release Instructions

This page covers instructions for releasing Argo Workflows.
It is intended for Argo release managers, who will be responsible for coordinating the release.
Release managers must be approvers on the Argo project.

## Patch Releases

Patch releases are for bug fixes and are released from an existing release branch.
Using the `cherry-pick` comment you can cherry-pick PRs from `main` to the release branch in advance of the release.
This is recommended to ensure that each PR is tested before it is published, and makes the process of releasing a new patch version much easier.
All members of the Argo project can cherry-pick fixes to release branches using this mechanism.

Manually raising cherry-pick PRs against a release branch is also acceptable, and can be done by anyone.

### Manually Cherry-Picking Fixes for patch releases

✋ Before you start, make sure you have created a release branch (e.g. `release-3.3`) and it's passing CI.
Please make sure that all patch releases (e.g. `v3.3.5`) should be released from their associated minor release branches (e.g. `release-3.3`) to work well with our versioned website.

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

Then look for "failed to cherry-pick" in the log to find commits that fail to be cherry-picked and decide if a manual patch is necessary.

Ignore:

* Fixes for features only on `main`.
* Dependency upgrades, unless they fix known security issues.
* Build or CI improvements, unless the release pipeline is blocked without them.

Cherry-pick the first commit.
Run `make test` locally before pushing.
If the build timeouts the build caches may have gone, try re-running.

Don't cherry-pick another commit until the CI passes.
It is harder to find the cause of a new failed build if the last build failed too.

Cherry-picking commits one-by-one and then waiting for the CI will take a long time.
Instead, cherry-pick each commit then run `make test` locally before pushing.

## Feature releases

If you're releasing a version of Argo where the minor or major version is changing, you're releasing a feature release and there is more work to do.
You must start with at least one release candidate.
See [Release Cycle](releases.md#release-cycle) for information about release candidates.

### Release candidates

For release candidates you should tag `main` for the release.
These take the form of 3.6.0-rc1 and the final digit increases for each RC.

### Documentation

Before or after the first release candidate you should ensure that [new-features.md](new-features.md) and [upgrading.md](upgrading.md) are updated.
A post should be made on a blog site (we usually use medium) announcing the release, and the new features.
This post should celebrate the new features and thank the contributors, including statistics from the release.

Post this blog post to the [Argo Workflows Contributors](https://cloud-native.slack.com/archives/C0510EUH90V) Slack channel and [Argo Maintainers](https://cloud-native.slack.com/archives/C022F03E6BD) Slack channel for comments.

Update these three items ([new-features.md](new-features.md), [upgrading.md](upgrading.md), blog post) for each release candidate and the final release.

### Final release

There should be no changes between the final release candidate and the actual release.
For the final release you should create a tag at the same place as the final release candidate.
You must also create a `release/<version>` branch from that same point.

Now you can add the branch to [readthedocs](https://app.readthedocs.org/projects/argo-workflows/) and then the new branch should be built and published.
Close the release candidate github issue and unpin it, and create a new issue patches to this branch.

### Expire old branches

Release n-2 is now out of support.
You should not delete anything to do with it.
Consider whether to do one final release for it.
Once that is done the old branch should be kept, but the pinned issue tracker issue should be unpinned and closed.
The readthedocs documentation build should be kept.

## Publish Release (all releases)

✋ Before you start, make sure the branch is passing CI.

Push a new tag to the release branch.
E.g.:

```bash
git tag v3.3.4
git push upstream v3.3.4 # or origin if you do not use upstream
```

GitHub Actions will automatically build and publish your release.
This takes about 1h.
Set yourself a reminder to check this was successful.

## Update Changelog (all releases)

Once the tag is published, GitHub Actions will automatically open a PR to update the changelog.
Once the PR is ready, you can approve it, enable auto-merge, and then run the following to force trigger the CI build:

```bash
git branch -D create-pull-request/changelog
git fetch upstream
git checkout --track upstream/create-pull-request/changelog
git commit -s --allow-empty -m "chore: Force trigger CI"
git push upstream create-pull-request/changelog
```

Once the changelog updates have been merged, you should announce on our Slack channels, [`#argo-workflows`](https://cloud-native.slack.com/archives/C01QW9QSSSK) and [`#argo-announcements`](https://cloud-native.slack.com/archives/C02165G1L48).
See [previous](https://cloud-native.slack.com/archives/C02165G1L48/p1701112932434469) [announcements](https://cloud-native.slack.com/archives/C01QW9QSSSK/p1701112957127489) as examples of what to write in the patch announcement.
