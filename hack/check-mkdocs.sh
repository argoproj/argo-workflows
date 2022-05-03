#!/usr/bin/env sh
set -eu

echo "Checking all docs are listed in mkdocs.yml..."

find docs -name '*.md' | sed 's|^docs/||' | while read -r f ; do
  if ! grep -Fq "$f" mkdocs.yml; then
    echo "❌ $f is missing from mkdocs.yml" >&2
    exit 1
  fi
done

echo "✅ Success - all docs appear to be listed in mkdocs.yml"