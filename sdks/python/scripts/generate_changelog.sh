#!/bin/bash

# @python_requires: gitchangelog
argo::generate::changelog() {
    : "${RELEASE_VERSION?Must define RELEASE_VERSION env variable}"

    local args="$@"
    local changelog=$(find ./ -type f -name 'CHANGELOG.*' -exec basename {} \;)

    gitchangelog $args > ${changelog}
}

args=("$@")

if [ "$0" = "$BASH_SOURCE" ] ; then
    >&2 echo -e "\nGenerating CHANGELOG... \n"
    argo::generate::changelog ${args[@]}
fi
