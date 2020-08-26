#!/bin/bash
set -eu -o pipefail

dir=$1
version=$2

find "$dir" -type f -name '*.yaml' | while read -r f ; do
  sed "s|version: latest|version: ${version}|" "$f" > .tmp
  mv .tmp "$f"
done
