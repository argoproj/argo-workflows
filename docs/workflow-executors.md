# Workflow Executors

A workflow executor is a process that conforms to a specific interface that allows Argo to perform certain actions like monitoring pod logs, collecting artifacts, managing container life-cycles, etc.

The executor to be used in your workflows can be changed in [the config map](./workflow-controller-configmap.yaml) under the `containerRuntimeExecutor` key (removed in v3.4).

## Emissary (emissary)

> v3.1 and after

Default in >= v3.3.

This is the most fully featured executor.

* Reliability:
    * Works on GKE Autopilot
    * Does not require `init` process to kill sub-processes.
* More secure:
    * No `privileged` access
    * Cannot escape the privileges of the pod's service account
    * Can [`runAsNonRoot`](workflow-pod-security-context.md).
* Scalable:
    * It reads and writes to and from the container's disk and typically does not use any network APIs unless resource
    type template is used.
* Artifacts:
    * Output artifacts can be located on the base layer (e.g. `/tmp`).
* Configuration:
    * `command` should be specified for containers.

You can determine values as follows:

```bash
docker image inspect -f '{{.Config.Entrypoint}} {{.Config.Cmd}}' argoproj/argosay:v2
```

[Learn more about command and args](https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#notes)

### Image Index/Cache

If you don't provide command to run, the emissary will grab it from container image. You can also specify it using the workflow spec or emissary will look it up in the **image index**. This is nothing more fancy than
a [configuration item](workflow-controller-configmap.yaml).

Emissary will create a cache entry, using image with version as key and command as value, and it will reuse it for specific image/version.

### Exit Code 64

The emissary will exit with code 64 if it fails. This may indicate a bug in the emissary.

## Docker (docker)

⚠️Deprecated. Removed in v3.4.

Default in <= v3.2.

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

⚠️Deprecated. Removed in v3.4.

* Secure
    * No `privileged` access
    * Cannot escape the privileges of the pod's service account
    * [`runAsNonRoot`](workflow-pod-security-context.md) - TBD, see [#4186](https://github.com/argoproj/argo-workflows/issues/4186)
* Scalable:
    * Operations performed against the local Kubelet
* Artifacts:
    * Output artifacts must be saved on volumes (e.g. [empty-dir](empty-dir.md)) and not the base image layer (e.g. `/tmp`)
* Step/Task result:
    * Warnings that normally goes to stderr will get captured in a step or a dag task's `outputs.result`. May require changes if your pipeline is conditioned on `steps/tasks.name.outputs.result`
* Configuration:
    * Additional Kubelet configuration maybe needed

## Kubernetes API (`k8sapi`)

⚠️Deprecated. Removed in v3.4.

* Reliability:
    * Works on GKE Autopilot
* Most secure:
    * No `privileged` access
    * Cannot escape the privileges of the pod's service account
    * Can [`runAsNonRoot`](workflow-pod-security-context.md)
* Least scalable:
    * Log retrieval and container operations performed against the remote Kubernetes API
* Artifacts:
    * Output artifacts must be saved on volumes (e.g. [empty-dir](empty-dir.md)) and not the base image layer (e.g. `/tmp`)
* Step/Task result:
    * Warnings that normally goes to stderr will get captured in a step or a dag task's `outputs.result`. May require changes if your pipeline is conditioned on `steps/tasks.name.outputs.result`
* Configuration:
    * No additional configuration needed.

## Process Namespace Sharing (`pns`)

⚠️Deprecated. Removed in v3.4.

* More secure:
    * No `privileged` access
    * cannot escape the privileges of the pod's service account
    * Can [`runAsNonRoot`](workflow-pod-security-context.md), if you use volumes (e.g. [empty-dir](empty-dir.md)) for your output artifacts
    * Processes are visible to other containers in the pod. This includes all information visible in /proc, such as passwords that were passed as arguments or environment variables. These are protected only by regular Unix permissions.
* Scalable:
    * Most operations use local `procfs`.
    * Log retrieval uses the remote Kubernetes API
* Artifacts:
    * Output artifacts can be located on the base layer (e.g. `/tmp`)
    * Cannot capture artifacts from a base layer which has a volume mounted under it
    * Cannot capture artifacts from base layer if the container is short-lived.
* Configuration:
    * No additional configuration needed.
* Process will no longer run with PID 1
* [Doesn't work for Windows containers](https://kubernetes.io/docs/setup/production-environment/windows/intro-windows-in-kubernetes/#v1-pod).

[Learn more](https://kubernetes.io/docs/tasks/configure-pod-container/share-process-namespace/)
