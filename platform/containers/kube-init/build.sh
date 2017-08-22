#!/bin/bash
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

set -xe

CONTAINER_DIR=`dirname $0`
SRCROOT=`dirname $0`/../../..
BUILD_DIR=`dirname $0`/docker_build

rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR
