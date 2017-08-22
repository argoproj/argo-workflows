#!/bin/bash

set -xe

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
. $DIR/../build_env.sh

gofmt -w $SRCROOT/saas/axops/src
go tool vet -composites=false $SRCROOT/saas/axops/src

#mkdir -p $SRCROOT/saas/axops/swagger/json
#echo -e "*\n*/\n!.gitignore\n" > $SRCROOT/saas/axops/swagger/json/.gitignore

if [ "$DEBUG" != "true" ]; then
   swagger -apiPackage="applatix.io/axops" -mainApiFile="applatix.io/axops/axops_server/main.go" -format go -ignore "github.com/*|context|golang.org*|k8s.io/*|cloud.google.com/*|git.apache.org/*|gopkg.in/*" -output $SRCROOT/saas/axops/src/applatix.io/axops/axops_server
   swagger -apiPackage="applatix.io/axops" -mainApiFile="applatix.io/axops/axops_server/main.go" -format swagger -ignore "github.com/*|context|golang.org*|k8s.io/*|cloud.google.com/*|git.apache.org/*|gopkg.in/*" -output $SRCROOT/saas/axops/swagger/json
fi

rm -rf $SRCROOT/saas/axops/bin

ARGO_VERSION=`grep . $SRCROOT/version.txt`
LD_FLAGS="-X applatix.io/axops/utils.Version=$ARGO_VERSION"

go install -v -ldflags "$LD_FLAGS" applatix.io/axops/axops_server
go install -v applatix.io/axops/axops_initializer
go install -v applatix.io/axops/axpassword

$SRCROOT/saas/argocli/build.sh

cp -rf $SRCROOT/saas/argocli/bin/argocli $SRCROOT/saas/axops/bin

# Copy UI static assets
rm -rf $SRCROOT/saas/axops/public/{assets/*,index.html,webpack-assets.json} && \
cp -r $SRCROOT/saas/axops/src/ui/dist/* $SRCROOT/saas/axops/public
