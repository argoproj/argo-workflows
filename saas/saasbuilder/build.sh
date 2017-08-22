#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
. $DIR/../build_env.sh

echo "Checking for previous version of $SAASBUILDER"
docker pull $SAASBUILDER
rc=$?
if [[ $rc == 0 ]]; then
    if [[ $1 != "force" ]]; then 
        echo "Someone has already built and pushed $SAASBUILDER. Either update the version in build_env.sh, or rerun with 'force' to overwrite previous pushed version"
        exit 1
    else
        echo "Overwriting previous image of $SAASBUILDER"
    fi
fi

set -e
cp $SRCROOT/saas/glide.yaml $SRCROOT/saas/saasbuilder
cp $SRCROOT/saas/glide.lock $SRCROOT/saas/saasbuilder
docker build -t $SAASBUILDER $SRCROOT/saas/saasbuilder && docker push $SAASBUILDER
