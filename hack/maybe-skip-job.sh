#!/bin/bash
set -eux -o pipefail

branch=$(git rev-parse --abbrev-ref=loose HEAD | sed 's/heads\///')

# always run on master
[ "$branch" = master ] && exit 0

fork_point=$(git merge-base --fork-point master)

skip_job() {
  set -eux -o pipefail
  circleci step halt
}

# do not run at all for docs only change
git diff --name-only "$fork_point" | grep -v '.circleci/\|.github/\|assets/\|community/\|docs/\|examples/\|hooks' || skip_job

case $1 in
codegen)
  git diff --name-only --exit-code "$fork_point" api manifests pkg || skip_job
  ;;
e2e)
  git diff --name-only --exit-code "$fork_point" manifests test/e2e/manifests '*.go' || skip_job
  ;;
test)
  git diff --name-only --exit-code "$fork_point" '*.go' || skip_job
  ;;
ui)
  git diff --name-only --exit-code "$fork_point" ui || skip_job
  ;;
esac
