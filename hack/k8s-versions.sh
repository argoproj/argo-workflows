#!/bin/bash
set -euo pipefail

# For the given revision, extract the Kubernetes versions tested in the
# corresponding e2e-tests CI workflow definition.
# This would be cleaner if we extracted the version data to a separate file,
# but this script is used to generate the "Tested versions" table, so it needs
# to be compatible with old release branches.
git grep -Eh 'INSTALL_K3S_VERSION=|install_k3s_version:' "${1:-HEAD}" -- .github/workflows/ci-build.yaml | grep -o 'v[0-9\.]\+' | sort -u
