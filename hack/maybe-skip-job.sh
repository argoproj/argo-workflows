#!/bin/bash
set -eux -o pipefail

branch=$(git rev-parse --abbrev-ref=loose HEAD | sed 's/heads\///')

# always run on master
[ "$branch" = master ] && exit

# do not run at all for docs only changes
if [ "$(git diff --name-only master | grep -v '.circleci/\|.github/\|assets/\|community/\|docs/\|examples/\|hooks')" = "" ]; then
  circleci step halt
  exit
fi

case $1 in
codegen)
  git diff --name-only --exit-code master api manifests pkg || circleci step halt
  ;;
e2e)
  git diff --name-only --exit-code master manifests test/e2e/manifests '*.go' || circleci step halt
  ;;
test)
  git diff --name-only --exit-code master '*.go' || circleci step halt
  ;;
ui)
  git diff --name-only --exit-code master ui || circleci step halt
  ;;
esac
