#!/usr/bin/env sh
set -eu

echo '# Changelog'
echo

tag=
# we skip v0.0.0 tags, so these can be used on branches without updating release notes
git tag -l 'v*' | grep -v 0.0.0 | sort -rd | while read last; do
  if [ "$tag" != "" ]; then
    echo "## $tag ($(git log $tag -n1 --format=%as))"
    echo
	  git --no-pager log --format=' * [%h](https://github.com/argoproj/argo-workflows/commit/%H) %s' $last..$tag
	  echo
	  echo "### Contributors"
	  echo
	  git --no-pager log --format=' * %an'  $last..$tag | sort -u
	  echo
  fi
  tag=$last
done