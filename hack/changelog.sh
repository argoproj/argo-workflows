#!/usr/bin/env sh
set -eu

echo '# Changelog'
echo

tag=
git tag -l 'v*' | sort -rd | while read last; do
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