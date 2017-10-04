#!/bin/bash
set -x
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
. $DIR/../build_env.sh

# Build UI static assets
rm -rf $DIR/src/ui/dist && mkdir -p $DIR/src/ui/dist && $DIR/src/ui/scripts/build.sh "$@"

build_in_container $DIR/build.sh
