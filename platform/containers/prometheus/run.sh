#!/bin/sh
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#
# Basic shell start script. Bash not supported.
#

set -ex

    if [ -z $MEM_MULT ] ; then
    echo "Must specify MEM_MULT"
    exit 1
fi

if [ -e /prometheus/crash_flag ]
then
    value=`cat /prometheus/crash_flag`
    cur=$(date +%s)
    difference=$(($cur-$value))
    threshold=300  # threshold is set to 5 min
    if [ $difference -le $threshold ]
    then
        # Clear previous data in the directory
        rm -rf /prometheus/data
    fi
fi

# Reset the time flag
echo $(date +%s) > /prometheus/crash_flag

# memory_persist is memory and should be small. Use 1/3 of memory_chunks.
#memory_chunks=`awk 'BEGIN{print int(60000*'$MEM_MULT');}'`
#memory_persist=`awk 'BEGIN{print int(20000*'$MEM_MULT');}'`

# target memory heap size is 1.0gb while there is requests is 1.2gb and limits is 3.0 gb allocated for prometheus (for small and medium cluster)
# retention set to 1 week

heap_target=`awk 'BEGIN{print int(1*1024*1024*1024*'$MEM_MULT');}'`
dirty_series_limit=`awk 'BEGIN{print int(1*1000*100*'$MEM_MULT');}'`

/bin/prometheus \
    -config.file=/etc/prometheus/prometheus.yml \
    -storage.local.target-heap-size=$heap_target \
    -storage.local.checkpoint-dirty-series-limit=$dirty_series_limit \
    -storage.local.retention=168h \
    -log.level=info
