#!/bin/bash

SRCROOT=`dirname $0`/../../
SRCROOT=`cd $SRCROOT;pwd`
source $SRCROOT/saas/build_env.sh

set -e

$SRCROOT/saas/axops/run-test-modules.sh
$SRCROOT/saas/axops/run-test-axops.sh
$SRCROOT/saas/axops/run-test-event.sh