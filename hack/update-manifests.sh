#!/bin/sh -x -e

SRCROOT="$( CDPATH='' cd -- "$(dirname "$0")/.." && pwd -P )"
AUTOGENMSG="# This is an auto-generated file. DO NOT EDIT"

IMAGE_NAMESPACE="${IMAGE_NAMESPACE:-argoproj}"
VERSION="${VERSION:-latest}"

cd ${SRCROOT}/manifests/base && kustomize edit set image \
    argoproj/workflow-controller=${IMAGE_NAMESPACE}/workflow-controller:${VERSION} \
    argoproj/argo-server=${IMAGE_NAMESPACE}/argo-server:${VERSION}

echo "${AUTOGENMSG}" > "${SRCROOT}/manifests/install.yaml"
kustomize build "${SRCROOT}/manifests/cluster-install" >> "${SRCROOT}/manifests/install.yaml"
sed -i.bak "s@- .*/argoexec:.*@- ${IMAGE_NAMESPACE}/argoexec:${VERSION}@" "${SRCROOT}/manifests/install.yaml"
rm -f "${SRCROOT}/manifests/install.yaml.bak"

echo "${AUTOGENMSG}" > "${SRCROOT}/manifests/namespace-install.yaml"
kustomize build "${SRCROOT}/manifests/namespace-install" >> "${SRCROOT}/manifests/namespace-install.yaml"
sed -i.bak "s@- .*/argoexec:.*@- ${IMAGE_NAMESPACE}/argoexec:${VERSION}@" "${SRCROOT}/manifests/namespace-install.yaml"
rm -f "${SRCROOT}/manifests/namespace-install.yaml.bak"
