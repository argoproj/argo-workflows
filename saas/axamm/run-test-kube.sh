#!/bin/bash

SRCROOT=`dirname $0`/../../
SRCROOT=`cd $SRCROOT;pwd`
source $SRCROOT/saas/build_env.sh

set -e

go test -v applatix.io/axamm/kube -timeout 30s $@
go test -v applatix.io/axamm/application -timeout 600s $@