#!/usr/bin/env bash
# Check the actual binary, not Argo's independently supplied linker version.
set -euo pipefail
trap 'echo "Binary build metadata does not match clean source: $binary" >&2' ERR
binary=${1:?Usage: verify-build-metadata.sh BINARY}
info=$(go version -m "$binary")
revision=$(git rev-parse HEAD)
field() { awk -v key="$1" '$1 == "build" && index($2, key "=") == 1 {sub("^[^=]*=", "", $2); print $2}' <<< "$info"; }
[[ $(field vcs) == git ]]
[[ $(field vcs.revision) == "$revision" ]]
[[ $(field vcs.modified) == false ]]
[[ $(field vcs.time) == "$(TZ=UTC git show -s --date=format-local:%Y-%m-%dT%H:%M:%SZ --format=%cd HEAD)" ]]
version=$(awk '$1 == "mod" && $2 == "github.com/argoproj/argo-workflows/v4" {print $3}' <<< "$info")
# Go chooses a matching release tag when present, otherwise a source-derived
# pseudo-version. Neither an absent version nor (devel) is acceptable.
[[ "$version" == v4.* && "$version" != *+dirty ]]
if git rev-parse --verify "refs/tags/$version^{commit}" >/dev/null 2>&1; then
    [[ $(git rev-parse "refs/tags/$version^{commit}") == "$revision" ]]
else
    timestamp=$(TZ=UTC git show -s --date=format-local:%Y%m%d%H%M%S --format=%cd HEAD)
    [[ "$version" == *"$timestamp-${revision:0:12}" ]]
fi
