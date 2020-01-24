#!/usr/bin/env bash
set -eu -o pipefail

SRCROOT="$( CDPATH='' cd -- "$(dirname "$0")/.." && pwd -P )"
IMAGE_NAMESPACE="${IMAGE_NAMESPACE:-argoproj}"
VERSION="${VERSION:-latest}"

if [[ $(echo "$VERSION" | cut -c 1-1) != 'v' ]]; then
  VERSION=latest
fi

cd ${SRCROOT}/manifests/base && kustomize edit set image \
    argoproj/workflow-controller=${IMAGE_NAMESPACE}/workflow-controller:${VERSION} \
    argoproj/argocli=${IMAGE_NAMESPACE}/argocli:${VERSION}

kustomize build "${SRCROOT}/manifests/cluster-install" | ${SRCROOT}/hack/auto-gen-msg.sh > "${SRCROOT}/manifests/install.yaml"
sed -i.bak "s@- .*/argoexec:.*@- ${IMAGE_NAMESPACE}/argoexec:${VERSION}@" "${SRCROOT}/manifests/install.yaml"
rm -f "${SRCROOT}/manifests/install.yaml.bak"

kustomize build "${SRCROOT}/manifests/namespace-install" | ${SRCROOT}/hack/auto-gen-msg.sh > "${SRCROOT}/manifests/namespace-install.yaml"
sed -i.bak "s@- .*/argoexec:.*@- ${IMAGE_NAMESPACE}/argoexec:${VERSION}@" "${SRCROOT}/manifests/namespace-install.yaml"
rm -f "${SRCROOT}/manifests/namespace-install.yaml.bak"


kustomize build ${SRCROOT}/manifests/quick-start/no-db | sed "s/:latest/:$VERSION/" | ${SRCROOT}/hack/auto-gen-msg.sh > ${SRCROOT}/manifests/quick-start-no-db.yaml
kustomize build ${SRCROOT}/manifests/quick-start/mysql | sed "s/:latest/:$VERSION/" | ${SRCROOT}/hack/auto-gen-msg.sh > ${SRCROOT}/manifests/quick-start-mysql.yaml
kustomize build ${SRCROOT}/manifests/quick-start/postgres | sed "s/:latest/:$VERSION/" | ${SRCROOT}/hack/auto-gen-msg.sh > ${SRCROOT}/manifests/quick-start-postgres.yaml

