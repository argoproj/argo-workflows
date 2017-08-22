#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
. $DIR/../build_env.sh

$PREFIX /usr/local/go/bin/gofmt -w $SRCROOT/saas
$PREFIX /usr/local/go/bin/go tool vet -composites=false $SRCROOT/saas/axnc/src/

$PREFIX rm -rf $SRCROOT/saas/axnc/bin

go install applatix.io/axnc/server
