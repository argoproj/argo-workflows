#!/bin/bash

# Centralized config to define the minimum and maximum tested Kubernetes versions.
# This is used in the CI workflow for e2e tests and the devcontainer
declare -A K8S_VERSIONS=(
  [min]=v1.30.9
  [max]=v1.32.1
)