#!/usr/bin/env bash

gotest() {
    PKG=$1
    TIMEOUT=$2
    if [ -z "$TIMEOUT" ]; then
        TIMEOUT=1200s
    fi
    if [ "$CODE_COVERAGE" = "false" ]; then
        go test -v ${PKG} -timeout ${TIMEOUT} -check.vv
    else
        go test -coverprofile ${SRCROOT}/saas/test/${PKG//\//-}.out -v ${PKG} -timeout ${TIMEOUT} -check.vv\
            && go tool cover -html=${SRCROOT}/saas/test/${PKG//\//-}.out -o ${SRCROOT}/saas/test/${PKG//\//-}.html
    fi
}
