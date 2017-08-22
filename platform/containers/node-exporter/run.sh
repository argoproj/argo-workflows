#!/bin/bash
# -*- coding: utf-8 -*-
#
# Copyright 2015-2016 Applatix, Inc. All rights reserved.
#

# We need static binary nothing to be on host for host mount.
# We don't have access to host on GKE. This is the easiest way to install nothing on host.
vendor=$(cat /rootfs/sys/class/block/sda/device/vendor)
if [ "${vendor:0:6}" = "Google" ] ; then
    curl -o /rootfs/etc/nothing https://s3-us-west-1.amazonaws.com/ax-public/nothing/nothing
    chmod +x /rootfs/etc/nothing
fi

/bin/node_exporter --collectors.enabled=diskstats,filesystem -collector.procfs /host/proc -collector.sysfs /host/sys -collector.filesystem.ignored-mount-points "^/(sys|proc|host|etc)($|/)"
