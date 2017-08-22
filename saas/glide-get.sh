#!/usr/bin/env bash

set -e

SRCROOT=`dirname $0`/../
SRCROOT=`cd $SRCROOT;pwd`

#curl https://glide.sh/get | sh

GOPATH=$SRCROOT/saas/common:$SRCROOT/saas/tests/luceneindex:$SRCROOT/saas/axdb:$SRCROOT/saas/axamm:$SRCROOT/saas/axops:$SRCROOT/saas/axnc:$SRCROOT/saas/argocli:$GOPATH

cd $SRCROOT/saas && ls -l && glide get $1
