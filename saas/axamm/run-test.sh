#!/bin/bash

SRCROOT=`dirname $0`/../../
SRCROOT=`cd $SRCROOT;pwd`
source $SRCROOT/saas/build_env.sh

source $SRCROOT/saas/test-helper.sh

set -e

gotest applatix.io/axamm/deployment
gotest applatix.io/axamm/application