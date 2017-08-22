#!/bin/bash

set -e

SRCROOT=`dirname $0`/../../../
SRCROOT=`cd $SRCROOT;pwd`

source $SRCROOT/saas/build_env.sh
# build using axdb-dev image first
docker run --rm=true -v $SRCROOT:/src -w /src/saas/tests/luceneindex -e "PATH=/usr/local/go/bin:/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:." -e DEV_USER=`whoami|cut -f1 -d'@'` -e DEV_UID=`id -u` -e DEV_GID=`id -g` $SAASBUILDER bash build.sh container
