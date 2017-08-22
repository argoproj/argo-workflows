#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
. $DIR/../build_env.sh

gofmt -w $SRCROOT/saas/axdb/src/
go tool vet -composites=false $SRCROOT/saas/axdb/src/

go install -v applatix.io/axdb/axdb_server

# copy SeedProvider for Cassandra
#$PREFIX chmod -R 777 $SRCROOT/saas/axdb/tools
#cd $SRCROOT/saas/axdb/tools/seedprovider && mvn package
cp /usr/share/cassandra/lib/kubernetes-cassandra.jar $SRCROOT/saas/axdb/kubernetes-cassandra.jar
