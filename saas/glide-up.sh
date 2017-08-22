#!/usr/bin/env bash

set -e

SRCROOT=`dirname $0`/../
SRCROOT=`cd $SRCROOT;pwd`

GOPATH=$SRCROOT/saas/common:$SRCROOT/saas/tests/luceneindex:$SRCROOT/saas/axdb:$SRCROOT/saas/axamm:$SRCROOT/saas/axops:$SRCROOT/saas/axnc:$SRCROOT/saas/argocli:$GOPATH

cd $SRCROOT/saas && ls -l && glide cache-clear && glide up --strip-vendor
