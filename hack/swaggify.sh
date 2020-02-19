#!/usr/bin/env bash

for f in $(find . -name '*swagger.json') ; do
    cat $f | sed 's/k8s.io/io.k8s/' | sed 's/github.com.argoproj.argo.pkg.apis.workflow.v1alpha1/io.argoproj.workflow.v1alpha/g' > tmp
    mv tmp $f
done