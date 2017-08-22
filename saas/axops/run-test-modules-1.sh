#!/bin/bash

SRCROOT=`dirname $0`/../../
SRCROOT=`cd $SRCROOT;pwd`
source $SRCROOT/saas/build_env.sh

source $SRCROOT/saas/test-helper.sh
set -e

gotest applatix.io/axops/index
gotest applatix.io/axops/cluster
gotest applatix.io/notification_center
gotest applatix.io/axops/project
gotest applatix.io/axops/sandbox 
gotest applatix.io/axops/fixture
gotest applatix.io/axops/tool 
gotest applatix.io/axops/policy 
gotest applatix.io/axops/service 
gotest applatix.io/axops/label

