#!/bin/bash

set -e

SRCROOT=`dirname $0`/..

VER=""
if [ -z "$1" ]
then
    echo "Version is not specified"
else
    VER=$1
    aws s3 sync "$SRCROOT/src/assets/docs" "s3://ax-public/docs/$VER" --grants read=uri=http://acs.amazonaws.com/groups/global/AllUsers --profile dev
fi
