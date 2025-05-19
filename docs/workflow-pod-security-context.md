# Workflow Pod Security Context

This document explains how to configure security context for workflow pods in Argo Workflows.

Running workflow pods as non-root is best practice for security.

You may need to do this if:

* Your cluster requires [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards), as enforced by [PSA](https://kubernetes.io/docs/concepts/security/pod-security-admission/) or other means.
* You need to comply with security policies that restrict root access.

## Basic Configuration

By default, all workflow pods run as root.

You can configure the [security context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/) for your workflow pod.

Here's a basic example that runs the pod as a non-root user:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: security-context-
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 8737 #; or any non-root user
```

!!! Warning "It is easy to make a workflow need root unintentionally"
    You may find that user's workflows have been written to require root with seemingly innocuous code. E.g. `mkdir /my-dir` would require root.

## Global Configuration

You can set these security context settings globally using [workflow defaults](default-workflow-specs.md).

## Non-root Executor Image

Argo provides a non-root executor image that runs by default as user 8737.

This is not the default executor for backwards compatibility, but using it is best practice.
Use this image when your security policies restrict pulling images that run as root.
You can run this image as root by specifying `runAsUser: 0`.

The image is available at `quay.io/argoproj/argoexec-nonroot:<version>`.

You can configure this as the default executor image in the [workflow-controller-configmap](workflow-controller-configmap.yaml):

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:
  executor: |
    image: quay.io/argoproj/argoexec-nonroot:<version>
```
