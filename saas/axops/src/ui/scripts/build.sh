#!/bin/bash

SRCROOT="$(cd "$(dirname "$0")/.." && pwd)"

docker build -t argo-ui-builder -f "$SRCROOT/Dockerfile.build" $SRCROOT && docker run --rm --user $(id -u $(whoami)):$(id -g $(whoami)) \
  -v $SRCROOT/dist:/src/dist \
  -v $SRCROOT/src:/src/src \
  -v $SRCROOT/config:/src/config \
  -v $SRCROOT/tsconfig.json:/src/tsconfig.json \
  -v $SRCROOT/tslint.json:/src/tslint.json -w /src -it argo-ui-builder npm run build:prod
