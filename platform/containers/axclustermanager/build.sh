#!/bin/bash
# -*- coding: utf-8 -*-
#
# Copyright 2015-2015 Applatix, Inc. All rights reserved.
#

set -xe

SRCROOT=`dirname $0`/../../..
BUILD_DIR=`dirname $0`/docker_build
DOCKER_SRC_DIR=$SRCROOT/platform/source/docker

rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR
cp $DOCKER_SRC_DIR/platform_src.sh $BUILD_DIR
cp -Rf $SRCROOT/platform/requirements $BUILD_DIR

$BUILD_DIR/platform_src.sh
cp -pr $SRCROOT/platform/cluster $BUILD_DIR/cluster

# Generate kubeup (cluster_install) version in cluster directory.
# It has similar format as ax version such as 2.1.1-a97a4bf[-dirty]
# except git hash is for cluster directory only.
kubeup_version=`cat $SRCROOT/version.txt`-`git -C $SRCROOT/platform/cluster log --pretty=format:'%h' . | head -1`
kubeup_diff=`git -C $SRCROOT/platform/cluster diff --shortstat .`
[[ -n "$kubeup_diff" ]] && kubeup_version="$kubeup_version-dirty"
echo $kubeup_version > $BUILD_DIR/cluster/version.txt
