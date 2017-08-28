#!/bin/bash

set -e

SRCROOT=`dirname $0`/../../
SRCROOT=`cd $SRCROOT;pwd`
source $SRCROOT/saas/build_env.sh

docker run --rm=true -i -e SRCROOT=$SRCROOT -v $SRCROOT:$SRCROOT -w $SRCROOT $SAASBUILDER sh -c '$SRCROOT/saas/common/config/cassandra-test-config.sh /etc/cassandra && $SRCROOT/saas/common/config/kafka-test-config.sh && $SRCROOT/saas/axamm/run-test.sh'
