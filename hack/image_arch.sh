#!/bin/sh
set -eu

if [ "${IMAGE_OS}" = "linux" ]; then
    export IMAGE_ARCH=`uname -m`;
    if [ "${IMAGE_ARCH}" = "x86_64" ]; then
        export IMAGE_ARCH=amd64;
    elif [ "${IMAGE_ARCH}" = "aarch64" ]; then
        export IMAGE_ARCH=arm64;
    fi
fi
