#!/usr/bin/env bash
set -eu -o pipefail
CLUSTER_NAME="${K3D_CLUSTER_NAME:-k3s-default}"
k3d cluster delete "${CLUSTER_NAME}"
