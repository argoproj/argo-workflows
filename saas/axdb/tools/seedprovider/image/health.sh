#!/bin/bash
# if version check failed, we will return true so that axdb-0/1/2 can move forward
RES=`curl --max-time 5 ${AXDB_SERVICE_HOST}:${AXDB_SERVICE_PORT}/v1/axdb/version | grep version | grep v1 | wc -l`
if [ "$RES" != "1" ]; then
   exit 0
fi

# if version check succeed, it means the cluster if forming, check if the current node is up
RES=`curl localhost:8080/v1/axdb/version | grep version | grep v1 | wc -l `
if [ "$RES" == "1" ]; then
   exit 0
else 
   exit 1
fi
#RES=`nodetool status | grep $POD_IP | grep "UN" | wc -l`
#if [ "$RES" == "1" ]; then
#   exit 0
#else
#   exit 1
#fi
