#!/bin/bash
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

SRCROOT=`dirname $0`/../../..
BUILD_DIR=`dirname $0`/docker_build

rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR
cp -Rf $SRCROOT/platform/requirements $BUILD_DIR
