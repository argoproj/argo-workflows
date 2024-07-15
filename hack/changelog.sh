#!/usr/bin/env sh
set -eu

# escape `*` with `\*`, but except first character
echo_escape() {
  echo "$@" | sed "s/\(.\\)[*]/\1\\\\*/g"
}

git_log='git --no-pager log --no-merges'
# exclude build, chore, ci, docs, and test of all scopes. always include deps scope and breaking changes. always exclude docs and deps-dev scopes and GHA dep bumps
# we use a denylist instead of an allowlist because of backward-compat: <=3.4.7 missing some conventional commits, <=2.5.0-rc missing most or all conventional commits
git_log_filtered="$git_log -P --invert-grep --grep=^(build|chore|ci|docs|test)(\((?!deps).*\))?: --invert-grep --grep=^(.+)(\(docs|deps-dev\)): --invert-grep --grep=^chore\(deps\):\sbump\s(actions|dependabot)/.*"

echo '# Changelog'

tag=
# we skip v0.0.0 tags, so these can be used on branches without updating release notes
git tag -l 'v*' | grep -v 0.0.0 | sed 's/-rc/~/' | sort -rV | sed 's/~/-rc/' | while read last; do
  if [ "$tag" != "" ]; then
    echo
    echo "## $(git for-each-ref --format='%(refname:strip=2) (%(creatordate:short))' refs/tags/${tag})"
    echo
    echo "Full Changelog: [$last...$tag](https://github.com/argoproj/argo-workflows/compare/$last...$tag)"
    echo
    echo "### Selected Changes"
    output=$($git_log_filtered --format='* [%h](https://github.com/argoproj/argo-workflows/commit/%H) %s' "$last..$tag")
    [ -n "$output" ] && echo && echo_escape "$output"
    echo
    echo "<details><summary><h3>Contributors</h3></summary>" # collapsed
    output=$($git_log --format='* %an' $last..$tag | sort -u)
    [ -n "$output" ] && echo && echo_escape "$output"
    echo
    echo "</details>"
  fi
  tag=$last
done
