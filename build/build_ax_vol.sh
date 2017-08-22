#!/bin/bash

function build-helper {
    make_opt=$1

    local src
    src=${PWD}
    cd $src/platform/source/go/docker/
    docker build -t ${ARGO_DEV_REGISTRY}/argobase/ax_vol_builder:latest .
    docker push ${ARGO_DEV_REGISTRY}/argobase/ax_vol_builder:latest
    cd $src

    echo ""
    echo "=== AX vol version: `git rev-parse --short HEAD` ==="
    echo ""
    docker run -v ~/.aws:/root/.aws \
        -v $src/platform/source/go/src/applatix.io:/source/platform/source/go/src/applatix.io \
        -v $src/platform/source/go/bin:/source/platform/source/go/bin ${ARGO_DEV_REGISTRY}/argobase/ax_vol_builder:latest \
        make version=`git rev-parse --short HEAD` $make_opt
}

if [ -z $1 ] || [ $1 == "--build" ]; then
      echo "Building AX volume"
      build-helper "all"
elif [ $1 == "--release" ]; then
      echo "Releasing AX volume"
      build-helper "release"
fi
