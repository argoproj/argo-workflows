#!/bin/bash

PREFIX=""

if [ "$1" == "container" ] ; then

    if [ x"$DEV_USER" = "x" -o x"$DEV_UID" = "x" -o x"$DEV_GID" = "x" ] ; then
        exit 1
    fi

    groupadd -g $DEV_GID $DEV_USER
    useradd -u $DEV_UID -g $DEV_GID -s /bin/bash $DEV_USER

    PREFIX="sudo -u $DEV_USER -E PATH=$PATH"
fi

set -e

SRCROOT=`dirname $0`/../../../
SRCROOT=`cd $SRCROOT;pwd`

$PREFIX /usr/local/go/bin/gofmt -w $SRCROOT/saas/tests/luceneindex/src/
$PREFIX /usr/local/go/bin/go tool vet $SRCROOT/saas/tests/luceneindex/src/

GOPATH=$GOPATH:$SRCROOT/saas/axdb:$SRCROOT/saas/axops:$SRCROOT/saas/common:$SRCROOT/saas/tests/luceneindex
$PREFIX /usr/local/go/bin/go install applatix.io/luceneindex/luceneindex_test
