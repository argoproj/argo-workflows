#!/usr/bin/env bash

# Centralized config to define the minimum and maximum tested Kubernetes versions.
# This is used in the CI workflow for e2e tests, the devcontainer, and to generate docs.
declare -A K8S_VERSIONS=(
  [min]=v1.33.10
  [max]=v1.35.0
)

# renovate: datasource=github-releases depName=k3d-io/k3d
K3D_VERSION=5.8.3
