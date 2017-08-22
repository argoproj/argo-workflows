#!/bin/bash

set -xe

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
. $DIR/../build_env.sh

gofmt -w $SRCROOT/saas/argocli/src

rm -rf $SRCROOT/saas/argocli/bin

ARGO_VERSION=`grep . $SRCROOT/version.txt`
ARGO_REVISION=`git -C $SRCROOT rev-parse --short=7 HEAD`
dirty=`git -C $SRCROOT diff --shortstat`
if [ ! -z "$dirty" ]; then
    ARGO_REVISION="${ARGO_REVISION}+"
fi

LD_FLAGS="-w -s -X applatix.io/argo/cmd.Version=$ARGO_VERSION -X applatix.io/argo/cmd.Revision=$ARGO_REVISION"

go install -v -ldflags "$LD_FLAGS" applatix.io/argo

if [ -f "/.dockerenv" ] ; then
    # if building in a docker container, assume we want to build for all platforms
    mkdir -p $SRCROOT/saas/argocli/bin/argocli/linux_amd64
    cp $SRCROOT/saas/argocli/bin/argo $SRCROOT/saas/argocli/bin/argocli/linux_amd64/argo
    if [ "$DEBUG" != "true" ] ; then
        # skip this if building a debug build, which we want to build quickly
        CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -v -ldflags "$LD_FLAGS" -a -installsuffix cgo -o $SRCROOT/saas/argocli/bin/argocli/darwin_amd64/argo applatix.io/argo
        CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -v -ldflags "$LD_FLAGS" -a -installsuffix cgo -o $SRCROOT/saas/argocli/bin/argocli/windows_amd64/argo.exe applatix.io/argo
    fi
    # build the argo.sh shell wrapper
    mkdir -p $SRCROOT/saas/argocli/bin/argocli/docker
    argo_sh_contents="#!/bin/sh\ndocker run -it --rm -v ~/.argo:/root/.argo argoproj/argocli:${ARGO_VERSION}-${ARGO_REVISION} \"\$@\""
    echo -e ${argo_sh_contents} > $SRCROOT/saas/argocli/bin/argocli/docker/argo.sh
    chmod ugo+x $SRCROOT/saas/argocli/bin/argocli/docker/argo.sh
fi

$SRCROOT/saas/argocli/bin/argo yaml validate $SRCROOT/.argo
