#!/bin/bash
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

set -e

BUILD_DIR=`dirname $0`
SRCROOT=$BUILD_DIR/../../../..
PLATFORM_SRC=$SRCROOT/platform
ROOT=$BUILD_DIR/root

# List of applictions which are bundled using PyInstaller.
# TODO: this list should be determined dynamically, perhaps by the 
# presence of a .spec file somewhere

BUNDLED_APPS="axmon.py axtool.py axconsole.py \
              axstats.py container_waiter.py \
              container_outer_executor.py \
              volume_mounts_fixer.py master_manager.py \
              minion_manager.py minion_upgrade.py \
              update_cluster_config.py upgrade-kubernetes.sh \
              search_subnet.py ax-upgrade-misc.py \
              managed_elb_creator.py \
              insert_builtin_templates.py managed_lb_upgrade.py axnotification.py"

# Ensure dirs
rm -rf $ROOT
mkdir -p $ROOT/ax/bin
mkdir -p $ROOT/ax/util

mkdir -p $ROOT/ax/config/cloud/compute
mkdir -p $ROOT/ax/config/cloud/standard

mkdir -p $ROOT/ax/config/service/standard
mkdir -p $ROOT/ax/config/service/mvc
mkdir -p $ROOT/ax/config/service/argo-wfe
mkdir -p $ROOT/ax/config/service/argo-gke
mkdir -p $ROOT/ax/config/service/argo-all
mkdir -p $ROOT/ax/config/service/config

mkdir -p $ROOT/ax/config/builtin-templates
mkdir -p $ROOT/ax/tests

# Copy ax libraries
cp -pRL $SRCROOT/common/python $ROOT/ax

# Copy cloud templates
cp $PLATFORM_SRC/config/cloud/compute/* $ROOT/ax/config/cloud/compute/
cp $PLATFORM_SRC/config/cloud/standard/* $ROOT/ax/config/cloud/standard/

# Copy applatix services
cp $PLATFORM_SRC/config/service/standard/* $ROOT/ax/config/service/standard/
cp $PLATFORM_SRC/config/service/mvc/* $ROOT/ax/config/service/mvc/
cp $PLATFORM_SRC/config/service/argo-wfe/* $ROOT/ax/config/service/argo-wfe/
cp $PLATFORM_SRC/config/service/argo-gke/* $ROOT/ax/config/service/argo-gke/
cp $PLATFORM_SRC/config/service/argo-all/* $ROOT/ax/config/service/argo-all/
cp $PLATFORM_SRC/config/service/config/* $ROOT/ax/config/service/config/

# Copy builtin templates
cp $PLATFORM_SRC/config/builtin-templates/* $ROOT/ax/config/builtin-templates


# copies the python files without extension
for script in $PLATFORM_SRC/source/tools/* ; do
    # Production builds will copy only single-file executables to the
    # container without any other dependencies. Debug builds will install
    # the python interpreter and copy our python source code into the
    # container. To support both production vs. debug containers
    # transparently, the python entrypoint scripts are copied without the 
    # .py extension, so that command  invocations are the same, regardless
    # if the application was bundled or not. 
    filename=$(basename "$script")
    if [[ ${filename} = "__pycache__" ]]; then
        continue
    elif [[ " $BUNDLED_APPS " =~ " ${filename} " ]]; then
        cp $script $ROOT/ax/bin/${filename%.*}
    else
        cp -r $script $ROOT/ax/bin/
    fi
done
cp $SRCROOT/bash-helpers.sh $PLATFORM_SRC/source/tools/common.sh $ROOT/ax/bin
cp $PLATFORM_SRC/tests/*.py $ROOT/ax/tests
mkdir -p $ROOT/root/.ssh
