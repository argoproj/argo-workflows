#!/usr/bin/env bash

set -eu

# Loosely based on https://github.com/argoproj/argo-cd/blob/5b79c34c72300e6e2e6336051ce6992f6d54011c/hack/update-supported-versions.sh

# Extract major/minor versions from branch name, e.g. "release-3.5" will become:
# release-
# 3
# 5
mapfile -t branch_parts < <(git rev-parse --abbrev-ref ${1:-HEAD} | grep -Eo '.*release-|[0-9]')

if [[ -z "${branch_parts[@]}" ]]; then
  echo 'This page is populated for released Argo Workflows versions. Use the version selector to view this table for a specific version.'
  exit
fi

echo "The following table shows the versions of Kubernetes that are tested with each version of Argo Workflows."
echo
echo "| Argo Workflows version | Kubernetes versions |"
echo "|------------------------|---------------------|"

for n in 0 1 2; do
  new_version="${branch_parts[1]}.$((branch_parts[2] - n))"
  new_branch="${branch_parts[0]}${new_version}"
  k8s_versions=$(./hack/k8s-versions.sh "$new_branch" | paste -s | sed 's/\t/, /')
  echo "|$new_version|$k8s_versions|"
done