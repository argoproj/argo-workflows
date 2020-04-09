#!/bin/bash
set -eux -o pipefail

branch=$(git rev-parse --abbrev-ref=loose HEAD | sed 's/heads\///')

# always run on master
[ "$branch" = master ] && exit

fork_point=$(git merge-base --fork-point master)

# do not run at all for docs only changes
if [ "$(git diff --name-only "$fork_point" | grep -v '.circleci/\|.github/\|assets/\|community/\|docs/\|examples/\|hooks')" = "" ]; then
  circleci step halt
  exit
fi

case $1 in
codegen)
  git diff --name-only --exit-code "$fork_point" api manifests pkg || circleci step halt
  ;;
e2e)
  git diff --name-only --exit-code "$fork_point" manifests test/e2e/manifests '*.go' || circleci step halt
  ;;
test)
  git diff --name-only --exit-code "$fork_point" '*.go' || circleci step halt
  ;;
ui)
  git diff --name-only --exit-code "$fork_point" ui || circleci step halt
  ;;
esac
