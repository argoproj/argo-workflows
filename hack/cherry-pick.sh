#!/usr/bin/env bash
set -eu
# See ../docs/releasing.md for instructions.

branch="$1" # branch name, e.g. release-3.3
commitPrefix="$2" # prefix to use to filter commits, e.g. fix, chore(deps), build, ci
# If dryRun is unset or `true`, only print the list of commits to be cherry-picked.
# Otherwise, cherry-pick the commits to the specified branch.
dryRun="${3:-"true"}"

commitGrepPattern="^${commitPrefix}(*.*)*:.*(#"

# find the branch point
base=$(git merge-base "$branch" main)

# extract the PRs from stdin
getPRNum() {
  cat | sed "s|.*(\(#[0-9]*\))|\1|"
}

# list the PRs on each branch
getPRs() {
  git log --format="%s" --grep "${commitGrepPattern}" "$1...$2" | getPRNum | sort > "/tmp/$2"
}

getPRs "$base" "$branch"
getPRs "$base" main

# find PRs added to main
diff "/tmp/$branch" /tmp/main | grep "^> " | cut -c 3- > /tmp/prs

# print all the commits that need cherry-picking
git log --oneline --grep "${commitGrepPattern}" "$base...main" | tac | while read -r m; do
  if grep -q "$(echo "$m" | getPRNum)" /tmp/prs ; then
    continue
  fi

  if [[ "$dryRun" == "true" ]]; then
    echo "$m"
    continue
  fi

  commit=${m:0:9}
  echo "cherry-picking: $commit"
  if ! git cherry-pick "$commit -x" ; then
    echo "failed to cherry-pick $commit"
    git cherry-pick --abort
  fi
done
