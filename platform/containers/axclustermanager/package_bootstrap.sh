#!/bin/bash

KUBE_ROOT="/kubernetes"
RELEASE_TMP="/tmp/release"
RELEASE_DIR="/kubernetes/server"

KUBE_MANIFEST="${RELEASE_DIR}/kubernetes-manifests.tar.gz"
KUBE_SALT="${RELEASE_DIR}/kubernetes-salt.tar.gz"
KUBE_BOOTSTRAP="${RELEASE_DIR}/bootstrap-script"


function log() {
    echo "[package-kubernetes-bootstrap] $1"
}


function create-tarball() {
    local tar_output=$1
    local tar_source=$2
    log "Creating tarball ${tar_output} ..."
    tar czf "${tar_output}" -C "${tar_source}" kubernetes --owner=0 --group=0
}


function clear-existing-build-cache() {
    rm -rf ${RELEASE_TMP}
    mkdir -p ${RELEASE_TMP}
    rm -f "${KUBE_MANIFEST}"
    rm -f "${KUBE_SALT}"
    rm -f "${KUBE_BOOTSTRAP}"
    log "Cleaned up all build caches"
}


function clean-cruft() {
    find $1 -name '*~' -exec rm {} \;
    find $1 -name '#*#' -exec rm {} \;
    find $1 -name '.DS*' -exec rm {} \;
}


# Involke kube-up's function to generate node bootstrap script for aws
function generate-aws-bootstrap-script() {
    log "Releasing kubernetes bootstrap script"
    export KUBE_TEMP=/kubernetes/server
    source /kubernetes/cluster/aws/util.sh
    create-bootstrap-script
    log "Successfully bootstrap script"
}


function release-salt() {
    log "Releasing Kubernetes salt ..."
    local release_stage="${RELEASE_TMP}/salt/kubernetes"
    mkdir -p "${release_stage}"
    cp -R "${KUBE_ROOT}/cluster/saltbase" "${release_stage}/"

    # (Harry) This logic is ported from build/common.sh in kubernetes v1.4.3 repo
    # we need to make sure we are doing the same thing
    # TODO(#3579): This is a temporary hack. It gathers up the yaml,
    # yaml.in, json files in cluster/addons (minus any demos) and overlays
    # them into kube-addons, where we expect them. (This pipeline is a
    # fancy copy, stripping anything but the files we don't want.)
    local objects
    objects=$(cd "${KUBE_ROOT}/cluster/addons" && find . \( -name \*.yaml -or -name \*.yaml.in -or -name \*.json \) | grep -v demo)
    tar c -C "${KUBE_ROOT}/cluster/addons" ${objects} | tar x -C "${release_stage}/saltbase/salt/kube-addons"

    clean-cruft "${release_stage}"
    create-tarball "${KUBE_SALT}" "${release_stage}/.."
    log "Successfully released Kubernetes salt"
}


# Main packaging routine
log "Start packaging Kubernetes bootstrap"
clear-existing-build-cache

generate-aws-bootstrap-script &
release-salt

wait
content_sum=$(sha1sum $KUBE_SALT $KUBE_BOOTSTRAP | awk '{print $1;}' | sha1sum)
sed -i 's/$/-'${content_sum:0:7}'/g' /kubernetes/cluster/version.txt

rm -rf "${RELEASE_TMP}"
