#!/bin/sh
# Convenience script to setup and enter the saas test environment from the current workspace,
# with all necessary environment variables set up to execute tests manually (using go test).

SRCROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )"
RANDPORT=$(($RANDOM+2001))
GOPATH="/go"
. $SRCROOT/saas/build_env.sh
echo "SAASBUILDER: $SAASBUILDER"
echo "GOPATH: $GOPATH"
echo "SRCROOT: $SRCROOT"
echo "Using random port range $RANDPORT-$((RANDPORT+5))"

docker run --rm --privileged -it -v $SRCROOT:$SRCROOT \
    -w $SRCROOT/saas \
    -p $RANDPORT:8080 -p $(( $RANDPORT+1 )):8081 -p $(( $RANDPORT+2 )):8082 -p $(( $RANDPORT+4 )):8084 -p $(( $RANDPORT+5 )):8085 \
    -e "PATH=$SRCROOT/prod/saas/common/bin:$SRCROOT/prod/saas/axevent/bin:$SRCROOT/prod/saas/axdb/bin:$SRCROOT/prod/saas/axops/bin:/usr/local/go/bin:/go/bin:$PATH" \
    -e "GOPATH=$GOPATH" \
    -e "SRCROOT=$SRCROOT" \
    --name=axdb-dev-$USER \
    $SAASBUILDER /bin/bash
