#!/usr/bin/env sh
set -eu

# Set locale for consistent sort output
export LC_ALL=en_US.UTF-8

# escape `*` with `\*` and then prepend each line with `* `
to_markdown_list() {
  sed -e 's/[*]/\\\\*/g' -e 's/^/* /'
}

git_log() {
  # exclude build, chore, ci, docs, and test of all scopes.
  # always include deps scope and breaking changes.
  # always exclude docs and deps-dev scopes and GHA dep bumps
  # we use a denylist instead of an allowlist because of backward-compat: <=3.4.7 missing some conventional commits, <=2.5.0-rc missing most or all conventional commits
  git log \
    --no-merges \
    --perl-regexp \
    --invert-grep '--grep=^(build|chore|ci|docs|test)(\((?!deps).*\))?:' \
    --invert-grep '--grep=^(.+)(\(docs|deps-dev\)):' \
    --invert-grep '--grep=^chore\(deps\):\sbump\s(actions|dependabot)/.*' \
    --format='[%h](https://github.com/argoproj/argo-workflows/commit/%H) %s' \
    "$1" | to_markdown_list
}

git_shortlog() {
    git shortlog --summary --group=author --group=trailer:co-authored-by "$1" | \
      grep -v '\[bot\]$' | \
      sed 's/^[0-9[:space:]]*//' | \
      sort -u | \
      to_markdown_list
}

echo '# Changelog'

tag=
# we skip v0.0.0 tags, so these can be used on branches without updating release notes
git tag -l 'v*' | grep -v 0.0.0 | sed 's/-rc/~/' | sort -rV | sed 's/~/-rc/' | while read last; do
  if [ "$tag" != "" ]; then
    echo
    echo "## $(git for-each-ref --format='%(refname:strip=2) (%(creatordate:short))' refs/tags/${tag})"
    echo
    echo "Full Changelog: [$last...$tag](https://github.com/argoproj/argo-workflows/compare/$last...$tag)"
    output=$(git_log "$last..$tag")
    if [ -n "$output" ]; then
      echo
      echo "### Selected Changes"
      echo
      echo "$output"
    fi
    output=$(git_shortlog "$last..$tag")
    if [ -n "$output" ]; then
      echo
      echo "<details><summary><h3>Contributors</h3></summary>" # collapsed
      echo
      echo "$output"
      echo
      echo "</details>"
    fi
  fi
  tag=$last

  # skip versions older than v3.x.x as those have been split into a separate file
  if [ "$tag" = "v2.12.13" ]; then
    break
  fi
done

# footer for versions older than 3.x.x
echo
echo "## v2.12.13 (2021-08-18)"
echo
echo "For v2.12.13 and earlier, see [CHANGELOG-2-x-x.md](CHANGELOG-2-x-x.md)"
