#!/usr/bin/env bash
set -eu -o pipefail

# Rewrite the SSO redirect URL to use HTTPS to support "make start PROFILE=sso UI_SECURE=true".
# Can't use "kubectl patch" because the SSO config is a YAML string.
kubectl -n "${KUBE_NAMESPACE}" get configmap workflow-controller-configmap -o yaml | \
    sed 's@redirectUrl: http://localhost:8080/oauth2/callback@redirectUrl: https://localhost:8080/oauth2/callback@' | \
    kubectl apply -n "${KUBE_NAMESPACE}" -f -