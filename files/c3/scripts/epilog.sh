#!/usr/bin/env bash

set -e

# Load libraries
. /c3/bin/libpackage.sh

install_package acl
install_package ca-certificates
install_package curl
install_package docker
install_package git
install_package gzip
install_package jq
install_package libcap
install_package procps
install_package tar

# Install kubectl
curl -o kubectl https://amazon-eks.s3.us-west-2.amazonaws.com/1.20.4/2021-04-12/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    mv ./kubectl /usr/local/bin/
