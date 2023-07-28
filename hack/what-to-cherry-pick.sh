#!/usr/bin/env sh
set -eu
# this script prints out a list a commits that are on master that you should probably cherry-pick to the release branch

br=$1;# branch
commitPrefix=$2;# examples: fix, chore(deps), build, ci
commitGrepPattern="^${commitPrefix}(*.*)*:.*(#"

# find the branch point
base=$(git merge-base $br master)

# extract the PRs from stdin
prNo() {
  set -eu
  cat | sed 's|.*(\(#[0-9]*\))|\1|'
}

# list the PRs on each branch
prs() {
  set -eu
  git log --format=%s --grep ${commitGrepPattern} $1...$2 | prNo | sort > /tmp/$2
}

prs $base $br
prs $base master

# find PRs added to master
diff /tmp/$br /tmp/master | grep '^> ' | cut -c 3- > /tmp/prs

# print all the commits that need cherry-picking
git log --oneline --grep ${commitGrepPattern} $base...master | while read -r m; do
  grep -q "$(echo $m | prNo)" /tmp/prs && echo $m
done