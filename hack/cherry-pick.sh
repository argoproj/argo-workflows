#!/usr/bin/env bash
set -eu
# See docs/releasing.md for instructions.

br="$1" # branch name, e.g. release-3.3
commitPrefix="$2" # prefix to use to filter commits, e.g. fix, chore(deps), build, ci
# whether this is dry-run. If set to `true`, this script will only print out the list
# of commits. Otherwise, this script will automatically cherry-pick the commits to the branch. 
dryRun="$3"

commitGrepPattern="^${commitPrefix}(*.*)*:.*(#"

# find the branch point
base=$(git merge-base "$br" master)

# extract the PRs from stdin
prNo() {
  set -eu
  cat | sed "s|.*(\(#[0-9]*\))|\1|"
}

# list the PRs on each branch
prs() {
  set -eu
  git log --format="%s" --grep "${commitGrepPattern}" "$1...$2" | prNo | sort > "/tmp/$2"
}

prs "$base" "$br"
prs "$base" master

# find PRs added to master
diff "/tmp/$br" /tmp/master | grep "^> " | cut -c 3- > /tmp/prs

# print all the commits that need cherry-picking
git log --oneline --grep "${commitGrepPattern}" "$base...master" | while read -r m; do
  if [[ "$dryRun" == "true" ]]; then
    grep -q "$(echo "$m" | prNo)" /tmp/prs && echo "$m"
  else
    commit=$(grep -q "$(echo "$m" | prNo)" /tmp/prs && echo "${m:0:9}")
    echo "cherry-picking: $commit"
    if ! git cp "$commit"; then
      echo "failed to cherry-pick $commit"
      git cp --abort
    fi
  fi
done
