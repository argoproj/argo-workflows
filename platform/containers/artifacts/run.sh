#!/usr/bin/env bash
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

set -x

if [ "$1" == "pre" ]
then
    echo "Copying AX bins to shared volume..."
    cp -r /ax-execu-host/art /copyto
    cp /usr/local/bin/docker ${AX_ARTIFACTS_SCRATCH}

    echo "Inspecting image..."
    docker inspect ${AX_CUSTOMER_IMAGE_NAME} > /inspect.txt
fi

echo "Generating service template env file"
echo ${AX_SERVICE_ENV} | base64 -d > /service.txt

if [ "$1" == "pre" ]
then
    echo "Generating executor script..."
   /ax/bin/container_outer_executor \
        --docker-inspect-result /inspect.txt \
        --host-scratch-root ${AX_ARTIFACTS_SCRATCH} \
        --container-scratch-root ${AX_ARTIFACTS_SCRATCH} \
        --executor-sh ${AX_ARTIFACTS_SCRATCH}/executor.sh \
        --pod-name ${AX_POD_NAME} \
        --job-name ${AX_JOB_NAME} \
        --pod-ip ${AX_POD_IP} \
        --input-label "in" --output-label "out"
else
    echo "Waiting for container to stop"
    /ax/bin/container_waiter ${AX_JOB_NAME} ${AX_POD_NAME} ${AX_MAIN_CONTAINER_NAME} ${AX_ARTIFACTS_SCRATCH} "out"
    /ax/bin/container_outer_executor \
        --docker-inspect-result /unused.txt \
        --host-scratch-root ${AX_ARTIFACTS_SCRATCH} \
        --container-scratch-root ${AX_ARTIFACTS_SCRATCH} \
        --executor-sh ${AX_ARTIFACTS_SCRATCH}/executor.sh \
        --input-label "in" --output-label "out" \
        --pod-name ${AX_POD_NAME} \
        --job-name ${AX_JOB_NAME} \
        --pod-ip ${AX_POD_IP} \
        --post-mode
    # Wait a little for debugging.
    # sleep 30
fi
