#!/bin/bash

# Pass in the argument that is the public port
if [ "$#" -ne 1 ]; then
    echo "Usage: ./run.sh public_port"
    exit 1
fi

base_registry=${ARGO_BASE_REGISTRY:-docker.io}
dev_registry=${ARGO_DEV_REGISTRY}

docker run -d -p $1:8080 -p $(( $1+1 )):8081 -p $(( $1+2 )):4000 -e "PATH=/axdb/bin:/usr/local/go/bin:/bin:/sbin:/usr/bin:/usr/sbin:." ${dev_registry}/$USER/axdb:latest
