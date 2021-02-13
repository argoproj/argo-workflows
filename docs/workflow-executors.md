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
* Equal most scalable:
    * It communicates directly with the local Docker daemon.
* Artifacts:
    * Output artifacts can be located on the base layer (e.g. `/tmp`).
* Configuration:
    * No additional configuration needed.

**Note**: when using docker as workflow executors, messages printed in both `stdout` and `stderr` are captured in the [Argo variable](./variables.md#scripttemplate) `.outputs.result`.

## Kubelet (kubelet)

* Reliability:
    * Second least well-tested
    * Second least popular
* Secure
    * No `privileged` access
    * Cannot escape the privileges of the pod's service account
    * [`runAsNonRoot`](workflow-pod-security-context.md) - TBD, see [#4186](https://github.com/argoproj/argo-workflows/issues/4186)
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
* Most secure:
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
* More secure:
    * No `privileged` access
    * cannot escape the privileges of the pod's service account
    * Can [`runAsNonRoot`](workflow-pod-security-context.md), if you use volumes (e.g. [emptyDir](empty-dir.md)) for your output artifacts
    * Processes are visible to other containers in the pod. This includes all information visible in /proc, such as passwords that were passed as arguments or environment variables. These are protected only by regular Unix permissions.
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

[https://kubernetes.io/docs/tasks/configure-pod-container/share-process-namespace/](https://kubernetes.io/docs/tasks/configure-pod-container/share-process-namespace/)

## Emissary (emissary)

![alpha](assets/alpha.svg)

> v3.1 and after

This is the most fully featured executor.

This executor works very differently to the others. It mounts an empty-dir on all containers at `/var/run/argo`. The main container command is replaces by a new binary `emissary` which starts the original command in a sub-process and when it is finished, captures the outputs:

The init container creates these files:

* `/var/run/argo/argoexec` The binary, copied from the `argoexec` image.
* `/var/run/argo/template` A JSON encoding of the template.

In the main container, the emissary creates these files: 

* `/var/run/argo/ctr/${containerName}/exitcode` The container exit code.
* `/var/run/argo/ctr/${containerName}/stderr` A copy of stderr. 
* `/var/run/argo/ctr/${containerName}/stdout`  A copy of stdout.

If the container is named `main` it also copies base-layer artifacts to the shared volume:

* `/var/run/argo/outputs/parameters/${path}` All output parameters are copied here, e.g. `/tmp/message` is moved to `/var/run/argo/outputs/parameters/tmp/message`.  
* `/var/run/argo/outputs/artifacts/${path}.tgz` All output artifacts are copied here, e.g. `/tmp/message` is moved to /var/run/argo/outputs/artifacts/tmp/message.tgz`.  

The wait container can create one file itself, used for terminating the sub-process:

* `/var/run/argo/ctr/${containerName}/signal` The emissary binary listens to changes in this file, and signals the sub-process with the value found in this file.

* Reliability:
  * Not yet well-tested.
  * Not yet popular.
* More secure:
  * No `privileged` access
  * Cannot escape the privileges of the pod's service account
  * Can [`runAsNonRoot`](workflow-pod-security-context.md).
* Scalable:
  * It reads and writes to and from the container's disk and typically does not use any network APIs unless resource type template is used.
* Artifacts:
  * Output artifacts can be located on the base layer (e.g. `/tmp`).
* Configuration:
  * `command` must be specified for containers. 
