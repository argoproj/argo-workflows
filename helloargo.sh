#!/usr/bin/env bash
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#


# User can specify one of the following actions:
#   - up        (Install an Argo cluster using the configurations)
#   - down      (Uninstall an Argo cluster using the configurations)
#   - pause     (Pause an Argo cluster)
#   - resume   (Restart an Argo cluster)
#   - upgrade   (Upgrade an Argo cluster)
#   - help      (Print help message)
# If it is kept blank, we will bring you into an Argo shell
ACTION=$1

# Command and cluster manager container related
CMD=""
DOCKER_ENV=""
DOCKER_VOL=""


####################################################################
# The following variables are configurations for software version
# They are for install and upgrade.
# Currently we don't support changing registry information during
# upgrade
####################################################################

# Specify where Argo software images should be pulled.
ARGO_DIST_REGISTRY=${ARGO_CLUSTER_DIST_REGISTRY:-"docker.io"}

# Base 64 encoded docker registry secret config. You don't need to specify
# it if the docker registry your image comes from is public
# For example, your docker config look like the follows:
#
# $ cat ~/.docker/config
# {
#     "auths": {
#         "docker.example.com": {
#             "auth": "EXAMPLE_AUTH_SECRET"
#         }
#     }
# }
#
# Encode this file with base 64 and set the environment variable:
#
# $ export ARGO_CLUSTER_DIST_REGISTRY_SECRETS=$(base64 -i ~/.docker/config)
ARGO_DIST_REGISTRY_SECRETS=${ARGO_CLUSTER_DIST_REGISTRY_SECRETS:-}


# Argo project software would following the following naming format:
# <registry>/<namespace>/image:<version>
AX_NAMESPACE=${ARGO_CLUSTER_IMAGE_NAMESPACE:-"argoproj"}
AX_VERSION=${ARGO_CLUSTER_IMAGE_VERSION:-"latest"}

############################### END OF CONFIGURATIONS #####################################


# Some useful colors
COLOR_RED="\033[0;31m"
COLOR_GREEN="\033[0;32m"
COLOR_YELLOW="\033[0;33m"
COLOR_NORM="\033[0m"


print-usage () {
    echo "Usage: $./helloargo.sh [up|down|pause|resume|upgrade|help]" >&2
}


validate-docker () {
    echo "Validating docker ..."
    if ! docker ps -a > /dev/null; then
        echo -e "${COLOR_RED}" >&2
        echo "    ERROR: Unable to connect to docker daemon. Is Docker running?" >&2
        echo -e "${COLOR_NORM}" >&2
        exit 1
    fi
}

ensure-dirs () {
    echo "Ensuring critical directories ..."
    mkdir -p ${HOME}/.kube
    mkdir -p ${HOME}/.argo
    mkdir -p ${HOME}/.ssh
}


set-docker-env () {
    DOCKER_ENV="-e AX_NAMESPACE=${AX_NAMESPACE} -e AX_VERSION=${AX_VERSION}"
    DOCKER_ENV="${DOCKER_ENV} -e AX_TARGET_CLOUD=${CLOUD_PROVIDER}"
    DOCKER_ENV="${DOCKER_ENV} -e ARGO_DIST_REGISTRY=${ARGO_DIST_REGISTRY}"
    DOCKER_ENV="${DOCKER_ENV} -e ARGO_DIST_REGISTRY_SECRETS=${ARGO_DIST_REGISTRY_SECRETS}"
}

set-docker-volumes () {
    DOCKER_VOL="-v ${HOME}/.aws:/root/.aws -v ${HOME}/.kube:/tmp/ax_kube -v ${HOME}/.ssh:/root/.ssh -v ${HOME}/.argo:/root/.argo"
}


if [[ "${ACTION}" == "help" ]]; then
    print-usage
    return 0
fi

validate-docker
ensure-dirs

if [[ -z "${ACTION}" ]]; then
    CMD="bash"
else
    CMD="argocluster ${ACTION}"
fi

echo "Running command:"
echo -e "${COLOR_GREEN}"
echo "    $ ${CMD}"
echo -e "${COLOR_NORM}"

set-docker-env
set-docker-volumes

echo -e "${COLOR_GREEN}"

if [[ -n "${ACTION}" ]]; then
    echo "Performing interactive operation \"${ACTION}\" on Argo Cluster ..."
else
    echo "Starting a shell to perform manual operations on Argo Cluster ..."

fi

echo -e "${COLOR_NORM}"
echo

docker pull ${ARGO_DIST_REGISTRY}/${AX_NAMESPACE}/axclustermanager:${AX_VERSION}
docker run -ti ${DOCKER_VOL} ${DOCKER_ENV} --net host ${ARGO_DIST_REGISTRY}/${AX_NAMESPACE}/axclustermanager:${AX_VERSION} ${CMD}
