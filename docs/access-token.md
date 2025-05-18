# Access Token

## Overview

If you want to automate tasks with the Argo Server API or CLI, you will need an access token.

## Prerequisites

Firstly, create a role with minimal permissions. This example role for jenkins only permission to update and list workflows:

```bash
kubectl create role jenkins --verb=list,update --resource=workflows.argoproj.io
```

Create a service account for your service:

```bash
kubectl create sa jenkins
```

### Tip for Tokens Creation

Create a unique service account for each client:

- (a) you'll be able to correctly secure your workflows
- (b) [revoke the token](#token-revocation) without impacting other clients.

Bind the service account to the role (in this case in the `argo` namespace):

```bash
kubectl create rolebinding jenkins --role=jenkins --serviceaccount=argo:jenkins
```

## Token Creation

You now need to create a secret to hold your token:

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: jenkins.service-account-token
  annotations:
    kubernetes.io/service-account.name: jenkins
type: kubernetes.io/service-account-token
EOF
```

Wait a few seconds:

```bash
ARGO_TOKEN="Bearer $(kubectl get secret jenkins.service-account-token -o=jsonpath='{.data.token}' | base64 --decode)"
echo $ARGO_TOKEN
Bearer ZXlKaGJHY2lPaUpTVXpJMU5pSXNJbXRwWkNJNkltS...
```

## Token Usage & Test

To use that token with the CLI you need to set `ARGO_SERVER` (see `argo --help`).

Use that token in your API requests, e.g. to list workflows:

```bash
curl https://localhost:2746/api/v1/workflows/argo -H "Authorization: $ARGO_TOKEN"
# 200 OK
```

You should check you cannot do things you're not allowed!

```bash
curl https://localhost:2746/api/v1/workflow-templates/argo -H "Authorization: $ARGO_TOKEN"
# 403 error
```

## Token Usage - Docker

### Set additional params to initialize Argo settings

```bash
ARGO_SERVER="${{HOST}}:443"
KUBECONFIG=/dev/null
ARGO_NAMESPACE=sandbox
```

### Start container with settings above

Example for listing templates in a namespace:

```bash
docker run --rm -it \
  -e ARGO_SERVER=$ARGO_SERVER \
  -e ARGO_TOKEN=$ARGO_TOKEN \
  -e ARGO_HTTP=false \
  -e ARGO_HTTP1=true \
  -e KUBECONFIG=/dev/null \
  -e ARGO_NAMESPACE=$ARGO_NAMESPACE  \
  quay.io/argoproj/argocli:latest template list -v -e -k
```

## Token Revocation

Token compromised?

```bash
kubectl delete secret $SECRET
```

A new one will be created.
