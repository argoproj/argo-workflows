#!/usr/bin/env sh
set -eu

# escape `*` with `\*`, but except first character
echo_escape() {
  echo "$@" | sed "s/\(.\\)[*]/\1\\\\*/g"
}

git_log='git --no-pager log --no-merges --invert-grep --grep=^\(build\|chore\|ci\|docs\|test\):'

echo '# Changelog'

tag=
# we skip v0.0.0 tags, so these can be used on branches without updating release notes
git tag -l 'v*' | grep -v 0.0.0 | sed 's/-rc/~/' | sort -rV | sed 's/~/-rc/' | while read last; do
  if [ "$tag" != "" ]; then
    echo
    echo "## $(git for-each-ref --format='%(refname:strip=2) (%(creatordate:short))' refs/tags/${tag})"
	  output=$($git_log --format='* [%h](https://github.com/argoproj/argo-workflows/commit/%H) %s' "$last..$tag")
    [ -n "$output" ] && echo && echo_escape "$output"
    echo
	  echo "### Contributors"
	  output=$($git_log --format='* %an' $last..$tag | sort -u)
    [ -n "$output" ] && echo && echo_escape "$output"
  fi
  tag=$last
done
