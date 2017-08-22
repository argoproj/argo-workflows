#!/bin/bash

SRCROOT=`dirname $0`/../../
SRCROOT=`cd $SRCROOT;pwd`
source $SRCROOT/saas/build_env.sh

source $SRCROOT/saas/test-helper.sh
set -e

#$SRCROOT/common/error/build.sh
gotest applatix.io/axdb
gotest applatix.io/axdb/core
