#!/bin/bash

SRCROOT="$(cd "$(dirname "$0")/.." && pwd)"

BUILD_CMD="build:prod"
if echo $* | grep -e "--debug" -q
then
  echo "Building UI in dev mode"
  BUILD_CMD="build:dev"
fi

docker build -t argo-ui-builder -f "$SRCROOT/Dockerfile.build" $SRCROOT && docker run --rm --user $(id -u $(whoami)):$(id -g $(whoami)) \
  -v $SRCROOT/dist:/src/dist \
  -v $SRCROOT/src:/src/src \
  -v $SRCROOT/config:/src/config \
  -v $SRCROOT/tsconfig.json:/src/tsconfig.json \
  -v $SRCROOT/tslint.json:/src/tslint.json -w /src -it argo-ui-builder npm run $BUILD_CMD
