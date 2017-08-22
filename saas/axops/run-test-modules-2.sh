#!/bin/bash

SRCROOT=`dirname $0`/../../
SRCROOT=`cd $SRCROOT;pwd`
source $SRCROOT/saas/build_env.sh

source $SRCROOT/saas/test-helper.sh
set -e

gotest applatix.io/axops/utils 
gotest applatix.io/axops/custom_view 
gotest applatix.io/axops/session 
gotest applatix.io/axops/user 
gotest applatix.io/axops/auth 
gotest applatix.io/axops/auth/native 
