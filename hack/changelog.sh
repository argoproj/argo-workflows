#!/usr/bin/env sh
set -eu

# escape `*` with `\*`, but except first character
echo_escape() {
  echo "$@" | sed "s/\(.\\)[*]/\1\\\\*/g"
}

git_log='git --no-pager log --no-merges'
git_log_filtered="$git_log --invert-grep --grep=^\(build\|chore\|ci\|docs\|test\):"

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
