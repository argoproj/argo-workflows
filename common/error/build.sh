#!/bin/bash

SRCROOT=`dirname $0`/../..

/bin/rm -rf $SRCROOT/common/error/gen*

# generated python is not used
#thrift -o $SRCROOT/common/error --gen py $SRCROOT/common/error/error.thrift 
thrift -o $SRCROOT/common/error --gen go $SRCROOT/common/error/error.thrift
thrift -o $SRCROOT/common/error --gen js $SRCROOT/common/error/error.thrift 

