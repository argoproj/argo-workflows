#!/usr/bin/env bash
# Create the k3d cluster used by Tilt for local dev and CI, if it does not
# already exist. The cluster name and k3s node image are overridable so CI can
# pin a Kubernetes version.
set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Use the centrally-defined supported Kubernetes versions (same source the CI
# e2e matrix uses), so dev and CI stay in lockstep. K8S_VERSION selects a key
# (min|max) or may be a literal version; K3D_K3S_IMAGE overrides the full ref.
# shellcheck source=/dev/null
. "${REPO_ROOT}/hack/k8s-versions.sh"

CLUSTER_NAME="${K3D_CLUSTER_NAME:-k3s-default}"
K8S_VERSION="${K8S_VERSION:-max}"
if [ -n "${K3D_K3S_IMAGE:-}" ]; then
  K3S_IMAGE="${K3D_K3S_IMAGE}"
else
  K8S_VER="${K8S_VERSIONS[$K8S_VERSION]:-$K8S_VERSION}"   # map min/max, else use literal
  K3S_IMAGE="rancher/k3s:${K8S_VER}-k3s1"
fi

# No image registry is needed: Tilt delivers images via `k3d image import`.
# k3d runs k3s with its embedded containerd (2.x in these k3s versions), which
# supports the Kubernetes image volumes the init-less pod layout uses to
# deliver argoexec; the imported images satisfy them because the e2e executor
# config sets imagePullPolicy: IfNotPresent.

if k3d cluster list "${CLUSTER_NAME}" >/dev/null 2>&1; then
  echo "k3d cluster '${CLUSTER_NAME}' already exists"
else
  args=(--wait)
  if [ -n "${K3S_IMAGE}" ]; then
    args+=(--image "${K3S_IMAGE}")
  fi
  # Stop the kubelet GCing images mid-test-run (which would be fatal: images
  # are delivered with `k3d image import`, so there is no registry to re-pull
  # from). See test/e2e/manifests/kubelet-configuration.yaml.
  args+=(--volume "${REPO_ROOT}/test/e2e/manifests/kubelet-configuration.yaml:/etc/rancher/k3s/kubelet.yaml@server:0")
  args+=(--k3s-arg "--kubelet-arg=config=/etc/rancher/k3s/kubelet.yaml@server:0")
  echo "creating k3d cluster '${CLUSTER_NAME}'"
  k3d cluster create "${CLUSTER_NAME}" "${args[@]}"
fi

# Wire up kubeconfig. k3d's built-in merge fails when KUBECONFIG lists multiple
# files, so when it does we write a dedicated kubeconfig file instead (one file
# per cluster). Add this file to your KUBECONFIG so future shells pick it up.
# With a single/unset KUBECONFIG (e.g. CI) we merge into the default kubeconfig.
if printf '%s' "${KUBECONFIG:-}" | grep -q ':'; then
  OUT="${K3D_KUBECONFIG_FILE:-$HOME/.kube/configs/k3d-${CLUSTER_NAME}.yaml}"
  mkdir -p "$(dirname "${OUT}")"
  k3d kubeconfig get "${CLUSTER_NAME}" > "${OUT}"
  echo "wrote kubeconfig to ${OUT}"
  echo "add it to your KUBECONFIG, e.g.: export KUBECONFIG=\"${OUT}:\$KUBECONFIG\""
  export KUBECONFIG="${OUT}:${KUBECONFIG}"
else
  k3d kubeconfig merge "${CLUSTER_NAME}" --kubeconfig-merge-default --kubeconfig-switch-context >/dev/null
fi

kubectl config use-context "k3d-${CLUSTER_NAME}"
kubectl cluster-info
