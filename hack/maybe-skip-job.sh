#!/bin/bash
set -eux -o pipefail

branch=$(git rev-parse --abbrev-ref=loose HEAD | sed 's/heads\///')

# always run on master
[ "$branch" = master ] && exit 0

fork_point=$(git merge-base --fork-point master)

skip_jobp() {
  set -eux -o pipefail
  circleci step halt
}

case $1 in
codegen)
  git diff --exit-code "$fork_point" api manifests pkg || skip_jobp
  ;;
ui)
  git diff --exit-code "$fork_point" ui || skip_jobp
  ;;
esac
