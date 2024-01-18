#!/usr/bin/env sh
set -eu

echo "Checking all docs are listed in mkdocs.yml..."ß

# https://www.mkdocs.org/user-guide/configuration/#validation since 1.5.0
mkdocs build --strict

echo "✅ Success - all docs appear to be listed in mkdocs.yml"
