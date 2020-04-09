#!/bin/bash
set -eux -o pipefail

branch=$(git rev-parse --abbrev-ref=loose HEAD | sed 's/heads\///')
job=$1
sha1=$CIRCLE_SHA1

# always run on master
[ "$branch" = master ] && exit

diffs=$(git diff --name-only master..$sha1)

if [ "$(echo $diffs | grep -v '.circleci/\|.github/\|assets/\|community/\|docs/\|examples/\|hooks')" = "" ]; then
  @echo "do not run at all for docs only changes"
  circleci step halt
  exit
fi

case $job in
codegen)
  echo "$diffs" | grep -v'api\|manifests\\pkg' || circleci step halt
  ;;
e2e)
  echo "$diffs" | grep -v 'manifests\|\.go' || circleci step halt
  ;;
test)
  echo "$diffs" | grep -v '\.go' || circleci step halt
  ;;
ui)
  echo "$diffs" | grep -v 'ui' || circleci step halt
  ;;
esac
