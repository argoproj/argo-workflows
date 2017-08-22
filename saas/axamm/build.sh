#!/bin/bash

set -xe

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
. $DIR/../build_env.sh

gofmt -w $SRCROOT/saas/axamm/src
go tool vet -composites=false $SRCROOT/saas/axamm/src/

go install applatix.io/axamm/axamm_server
go install applatix.io/axamm/axam_server
