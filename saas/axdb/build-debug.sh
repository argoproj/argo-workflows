#!/bin/bash

SRCROOT=`dirname $0`/../../
SRCROOT=`cd $SRCROOT;pwd`
source $SRCROOT/saas/build_env.sh

# build error codes
$SRCROOT/common/error/build.sh

gofmt -w $SRCROOT/saas/axdb
go tool vet $SRCROOT/saas/axdb

$SRCROOT/saas/common/bin/godebug  build -instrument applatix.io/axdb,applatix.io/axdb/core -o bin/axdb_server_debug applatix.io/axdb/axdb_server


