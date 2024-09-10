# Workflow Executors

A workflow executor is a process that conforms to a specific interface that allows Argo to perform certain actions like monitoring pod logs, collecting artifacts, managing container life-cycles, etc.

## Emissary (emissary)

> v3.1 and after

Default in >= v3.3.
Only option in >= v3.4.

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
