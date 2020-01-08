#!/bin/bash

CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
GIT_COMMIT=`git rev-parse --short HEAD`
VERSION=`cat ${CURRENT_DIR}/../VERSION`

set -e

TAG=${IMAGE_TAG:-"$VERSION-$GIT_COMMIT"}

docker build --build-arg ARGO_VERSION=${TAG} -t ${IMAGE_NAMESPACE:-`whoami`}/argoui:${TAG} .

if [ "$DOCKER_PUSH" == "true" ]
then
    docker push ${IMAGE_NAMESPACE:-`whoami`}/argoui:${TAG}
fi
