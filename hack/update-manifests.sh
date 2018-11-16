#!/bin/sh

IMAGE_NAMESPACE=${IMAGE_NAMESPACE:='argoproj'}
IMAGE_TAG=${IMAGE_TAG:='latest'}

autogen_warning="# This is an auto-generated file. DO NOT EDIT"

echo $autogen_warning > manifests/install.yaml
kustomize build manifests/cluster-install >> manifests/install.yaml

echo $autogen_warning > manifests/namespace-install.yaml
kustomize build manifests/namespace-install >> manifests/namespace-install.yaml
