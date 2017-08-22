#!/bin/bash

# Pass in the argument that is the root of the git repo
if [ "$#" -ne 1 ]; then
    echo "You must enter the root of git repo as the argument"
    exit 1
fi

set -e

SRCROOT=`dirname $0`/../../
SRCROOT=`cd $SRCROOT;pwd`
source $SRCROOT/saas/build_env.sh

docker run --rm=true -i -v $1/prod:/prod -w /prod $SAASBUILDER sh -c '/prod/saas/common/config/cassandra-test-config.sh /etc/cassandra && /prod/saas/common/config/kafka-test-config.sh && /prod/saas/axamm/run-test.sh'
