#!/bin/bash
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

set -ex

apt-get update && apt-get install -y \
    curl \
    python \
    python-dev \
    python3 \
    python3-dev \
    wget \
    ssh \
    git \
    vim \
    python-pip

pip install -r /tmp/requirements.txt
pip install awscli

curl -fSL "https://${DOCKER_BUCKET}/builds/Linux/x86_64/docker-$DOCKER_VERSION.tgz" -o docker.tgz \
    && echo "${DOCKER_SHA256} *docker.tgz" | sha256sum -c - \
    && tar -xzvf docker.tgz \
    && mv docker/docker /usr/local/bin/ \
    && rm -rf docker \
    && rm docker.tgz

mkdir -p /kubernetes/server/

curl -o /tmp/google.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-${GOOGLE_CLOUD_SDK_VERSION}-linux-x86_64.tar.gz && tar axvf /tmp/google.tar.gz -C /opt && rm /tmp/google.tar.gz
/opt/google-cloud-sdk/bin/gcloud -q components install alpha beta

mkdir -p /opt/google-cloud-sdk/bin
for ver in ${KUBERNETES_VERSIONS_LIST} ; do
    curl -L -o /opt/google-cloud-sdk/bin/kubectl-${ver} https://storage.googleapis.com/kubernetes-release/release/v${ver}/bin/linux/amd64/kubectl && \
        chmod u+x /opt/google-cloud-sdk/bin/kubectl-${ver} && \
        strip -s /opt/google-cloud-sdk/bin/kubectl-${ver}
done

apt-get purge -y python-dev python3-dev git gcc
apt-get autoremove -y
apt-get clean -y
for py in /usr/lib/python2.7 /usr/lib/python3.5 /usr/local/lib/python2.7 /usr/local/lib/python3.5 /opt/google-cloud-sdk ; do
    find $py -name "*.pyc" | xargs -r rm
done
rm -rf /root/.cache
rm -rf /opt/google-cloud-sdk/.install
rm -rf /var/lib/apt/lists/* && rm -rf /var/cache/apt
rm -rf /usr/share/man /usr/share/locale /usr/share/i18n /usr/share/doc
