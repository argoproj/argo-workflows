#!/bin/bash
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

# Run this build script directly without using build_platform.py.

cp -Rf `dirname $0`/../../../platform/requirements/requirements.txt .

# Configure kubernetes-distro version.
k8s_ver="v1.6.7"
ver="ax-${k8s_ver}"
seq="6341"
commit=`git ls-remote https://github.com/kubernetes/kubernetes.git |grep refs/tags/${k8s_ver} | sed -n 2p | cut -f 1 | cut -c1-7`
machine=`uname -s`
# if md5sum is null, reset dirty to null.
docker_version="$ver-$seq-$commit"

if [[ $machine == "Linux" ]]; then
    all_hash=$(echo $docker_version | sha1sum)
else
    all_hash=$(echo $docker_version | shasum)
fi

echo $ver
echo $seq
echo $commit
echo $all_hash

echo "Creating ax-kubernetes build with version: $docker_version-${all_hash:0:7}"
docker build -t ${ARGO_DEV_REGISTRY}/paralus/kubernetes-distro:$docker_version-${all_hash:0:7} .

