# Workflow Controller Configmap

## Introduction

The Workflow Controller Configmap is used to set controller-wide settings.

For a detailed example, please see [`workflow-controller-configmap.yaml`](./workflow-controller-configmap.yaml).

## Setting the Configmap

The configmap should be saved as a K8s Configmap on the cluster in the same namespace as the `workflow-controller`.
It should then be referenced by the `workflow-controller` as an command argument:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: workflow-controller
spec:
  selector:
    matchLabels:
      app: workflow-controller
  template:
    metadata:
      labels:
        app: workflow-controller
    spec:
      containers:
      - args:
        - --configmap
        - workflow-controller-configmap   # Set configmap name here
        - --executor-image
        - argoproj/argoexec:latest
        - --namespaced
        command:
        - workflow-controller
        image: argoproj/workflow-controller:latest
        name: workflow-controller
      serviceAccountName: argo
      nodeSelector:
              kubernetes.io/os: linux
```

## Alternate Structure

In all versions, the configuration may be under a `config: |` key:

```yaml
# This file describes the config settings available in the workflow controller configmap
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:
  config: |
    instanceID: my-ci-controller
    artifactRepository:
      archiveLogs: true
      s3:
        endpoint: s3.amazonaws.com
        bucket: my-bucket
        region: us-west-2
        insecure: false
        accessKeySecret:
          name: my-s3-credentials
          key: accessKey
        secretKeySecret:
          name: my-s3-credentials
          key: secretKey

```

In version 2.7+, the `config: |` key is optional. However, if the `config: |` key is not used, all nested maps under top level
keys should be strings. This makes it easier to generate the map with some configuration management tools like Kustomize.

```yaml
# This file describes the config settings available in the workflow controller configmap
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:                      # "config: |" key is optional in 2.7+!
  instanceID: my-ci-controller
  artifactRepository: |    # However, all nested maps must be strings
   archiveLogs: true
   s3:
     endpoint: s3.amazonaws.com
     bucket: my-bucket
     region: us-west-2
     insecure: false
     accessKeySecret:
       name: my-s3-credentials
       key: accessKey
     secretKeySecret:
       name: my-s3-credentials
       key: secretKey
```
