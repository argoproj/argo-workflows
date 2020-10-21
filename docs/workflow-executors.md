# Workflow Executors

A workflow executor is a process that conforms to a specific interface that allows Argo to perform certain actions like monitoring pod logs, collecting artifacts, managing container lifecycles, etc..

The executor to be used in your workflows can be changed in [the configmap](./workflow-controller-configmap.yaml) under the `containerRuntimeExecutor` key.


## Docker (docker)

**default**

* Reliability:
    * Most well-tested
    * Most popular
* Least secure:
    * It requires `privileged` access to `docker.sock` of the host to be mounted which. Often rejected by Open Policy Agent (OPA) or your Pod Security Policy (PSP).
    * It can escape the privileges of the pod's service account
    * It cannot [`runAsNonRoot`](workflow-pod-security-context.md).
* Most scalable:
    * It communicates directly with the local Docker daemon.
* Artifacts:
    * Output artifacts can be located on the base layer (e.g. `/tmp`).
* Configuration:
    * No additional configuration needed.

## Kubelet (kubelet)

* Reliability:
    * Least well-tested
    * Least popular
* Secure
    * No `privileged` access
    * Cannot escape the privileges of the pod's service account
    * [`runAsNonRoot`](workflow-pod-security-context.md) - TBD, see [#4186](https://github.com/argoproj/argo/issues/4186)
* Scalable:
    * Operations performed against the local Kubelet
* Artifacts:
    * Output artifacts must be saved on volumes (e.g. [emptyDir](empty-dir.md)) and not the base image layer (e.g. `/tmp`)
* Configuration:
    * Additional Kubelet configuration maybe needed

## Kubernetes API (k8sapi)

* Reliability:
    * Well-tested
    * Popular
* Secure:
    * No `privileged` access
    * Cannot escape the privileges of the pod's service account
    * Can [`runAsNonRoot`](workflow-pod-security-context.md)
* Least scalable:
    * Log retrieval and container operations performed against the remote Kubernetes API
* Artifacts:
    * Output artifacts must be saved on volumes (e.g. [emptyDir](empty-dir.md)) and not the base image layer (e.g. `/tmp`)
* Configuration:
    * No additional configuration needed.

## Process Namespace Sharing (pns)

* Reliability:
    * Well-tested
    * Popular
* Secure:
    * No `privileged` access
    * cannot escape the privileges of the pod's service account
    * Can [`runAsNonRoot`](workflow-pod-security-context.md), if you use volumes (e.g. [emptyDir](empty-dir.md)) for your output artifacts
* Scalable:
    * Most operations use local `procfs`.
    * Log retrieval uses the remote Kubernetes API
* Artifacts:
    * Output artifacts can be located on the base layer (e.g. `/tmp`)
    * Cannot capture artifacts from a base layer which has a volume mounted under it
* Configuration:
    * No additional configuration needed.
* Process will no longer run with PID 1
* [Doesn't work for Windows containers](https://kubernetes.io/docs/setup/production-environment/windows/intro-windows-in-kubernetes/#v1-pod).

