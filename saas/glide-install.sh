#!/usr/bin/env bash

set -e

SRCROOT=`dirname $0`/../
SRCROOT=`cd $SRCROOT;pwd`

#curl https://glide.sh/get | sh

GOPATH=$SRCROOT/saas/common:$SRCROOT/saas/common/vendor:$SRCROOT/saas/tests/luceneindex:$SRCROOT/saas/axdb:$SRCROOT/saas/axamm:$SRCROOT/saas/axops:$SRCROOT/saas/axnc:$SRCROOT/saas/argocli:$GOPATH

echo $GOPATH

cd $SRCROOT/saas && ls -l && glide install && rm -rf  $SRCROOT/saas/common/vendor/src && mv vendor src && mv src $SRCROOT/saas/common/vendor
