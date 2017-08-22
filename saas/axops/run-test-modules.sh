#!/bin/bash

SRCROOT=`dirname $0`/../../
SRCROOT=`cd $SRCROOT;pwd`
source $SRCROOT/saas/build_env.sh

set -e

$SRCROOT/saas/axops/run-test-modules-1.sh
$SRCROOT/saas/axops/run-test-modules-2.sh
