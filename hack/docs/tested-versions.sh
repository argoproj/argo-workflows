#!/usr/bin/env bash
set -eu
. hack/k8s-versions.sh
printf 'This version is tested under Kubernetes %s and %s.' "${K8S_VERSIONS[min]}" "${K8S_VERSIONS[max]}"