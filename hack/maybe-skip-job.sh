#!/bin/bash
set -eux -o pipefail

branch=$(git rev-parse --abbrev-ref=loose HEAD | sed 's/heads\///')
job=$1

# always run on master
[ "$branch" = master ] && exit

# tip - must use origin/master for CircleCI
diffs=$(git diff --name-only origin/master)

# if there are changes to this areas, we must run
rx=
case $job in
codegen)
  rx='api/\|hack/\|manifests/\|pkg/'
  ;;
docker-build)
  # we only run on master as this rarely ever fails
  circleci step halt
  exit
  ;;
e2e-*)
  rx='manifests/\|\.go'
  ;;
test)
  rx='\.go'
  ;;
ui)
  rx='ui/'
  ;;
esac

if [ "$(echo "$diffs" | grep "$rx")" = "" ]; then
  circleci step halt
  exit
fi
