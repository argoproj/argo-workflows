#!/bin/bash
# -*- coding: utf-8 -*-
#
# Copyright 2015-2015 Applatix, Inc. All rights reserved.
#

set -xe

CONTAINER_DIR=`dirname $0`
SRCROOT=`dirname $0`/../../..
BUILD_DIR=`dirname $0`/docker_build
DOCKER_SRC_DIR=$SRCROOT/platform/source/docker

rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR
cp $DOCKER_SRC_DIR/platform_src.sh $BUILD_DIR
cp -Rf $SRCROOT/platform/requirements $BUILD_DIR
$BUILD_DIR/platform_src.sh
