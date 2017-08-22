#!/bin/sh

set -xe

services="flower:5555 kibana:5601 redis:6379 jenkins:8082 axdb:8083 gateway:8889 notdefiend:9999"

for svc in $services ; do
    host=`echo $svc | awk -F':' '{print $1;}'`
    port=`echo $svc | awk -F':' '{print $2;}'`
    socat TCP-LISTEN:$port,fork,reuseaddr TCP:$host:$port &
done

wait
