#!/usr/bin/env bash

if [[ -z $SRCROOT ]] ; then
    SRCROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )"
fi

base_registry=${ARGO_BASE_REGISTRY:-docker.io}
dev_registry=${ARGO_DEV_REGISTRY}

export SAASBUILDER="${base_registry}/argobase/saasbuilder:v1"

# GOPATH sequence matters here, common contains customized libraries, common/vendor must be after the common
if [[ -z $GOPATH ]] ; then
    export GOPATH=$SRCROOT/saas/common:$SRCROOT/saas/common/vendor:$SRCROOT/saas/tests/luceneindex:$SRCROOT/saas/axdb:$SRCROOT/saas/axamm:$SRCROOT/saas/axops:$SRCROOT/saas/axnc:$SRCROOT/saas/argocli
else
    export GOPATH=$SRCROOT/saas/common:$SRCROOT/saas/common/vendor:$SRCROOT/saas/tests/luceneindex:$SRCROOT/saas/axdb:$SRCROOT/saas/axamm:$SRCROOT/saas/axops:$SRCROOT/saas/axnc:$SRCROOT/saas/argocli:${GOPATH}
fi

build_in_container() {
    USER="$(id -u):$(id -g)"
    docker run --rm=true -v $SRCROOT:$SRCROOT --user $USER $SAASBUILDER $1
}
