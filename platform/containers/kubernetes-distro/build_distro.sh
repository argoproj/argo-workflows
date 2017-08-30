#!/bin/bash
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

# Run this build script directly without using build_platform.py.

cp -Rf `dirname $0`/../../../platform/requirements/requirements.txt .

export KUBERNETES_VERSION=1.6.7
docker build -t ${ARGO_DEV_REGISTRY}/paralus/kubernetes-distro:v${KUBERNETES_VERSION} .

