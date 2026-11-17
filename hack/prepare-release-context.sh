#!/usr/bin/env bash
# Export committed source with Git metadata, without caller credentials or hooks.
set -euo pipefail
source_dir=$(git rev-parse --show-toplevel)
revision=$(git rev-parse HEAD)
if [[ $(git rev-parse --is-shallow-repository) == true ]]; then
    echo 'Release builds require complete history and tags' >&2
    exit 1
fi
destination=${1:?Usage: prepare-release-context.sh NEW_DIRECTORY}
if [[ -n $(git status --porcelain --untracked-files=no) ]]; then
    echo 'Release builds require clean tracked source' >&2
    exit 1
fi
if git ls-files --stage | grep '^160000 ' >/dev/null; then
    echo 'Release contexts do not support submodules' >&2
    exit 1
fi
mkdir "$destination"
destination=$(cd "$destination" && pwd)
# Avoid copying .git/config (including checkout credentials), hooks or alternates.
GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null git init --template= "$destination/source"
# Keep ancestry and tags so untagged builds derive their pseudo-version from
# the actual preceding release, rather than falling back to v4.0.0.
git -C "$destination/source" -c core.hooksPath=/dev/null -c protocol.file.allow=always \
    fetch --tags --no-recurse-submodules "$source_dir" "$revision"
git -C "$destination/source" -c core.hooksPath=/dev/null checkout --detach FETCH_HEAD
[[ $(git -C "$destination/source" rev-parse HEAD) == "$revision" ]]
[[ -z $(git -C "$destination/source" status --porcelain) ]]
# Docker's ordinary ignore list drops tracked files and .git, making VCS
# stamping impossible. Override it only for this clean, committed snapshot.
cp "$destination/source/Dockerfile" "$destination/Dockerfile"
printf '!**\n' > "$destination/Dockerfile.dockerignore"
