# Workflow Executors

A workflow executor runs in the pods that execute your workloads.
It runs as both an init container and as a sidecar to the container you specify.
It allows Argo to perform certain actions like monitoring pod logs, providing and collecting artifacts, and managing container life-cycles.

Historically there were multiple available executor types, but as of 3.4 the only available executor is called `emissary`.

## Emissary Executor

* Reliability:
    * Works on GKE Autopilot
    * Does not require `init` process to kill sub-processes
* Security:
    * No `privileged` access
    * Cannot escape the privileges of the pod's service account
    * Supports [running as non-root](workflow-pod-security-context.md)
* Scalability:
    * Reads and writes to and from the container's disk
    * Typically does not use network APIs unless resource type template is used
* Artifacts:
    * Output artifacts can be located on the base layer (e.g. `/tmp`)

### Container Command

You can determine the default command for a container image using:

```bash
docker image inspect -f '{{.Config.Entrypoint}} {{.Config.Cmd}}' argoproj/argosay:v2
```

[Learn more about command and args](https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#notes)

### Image Index/Cache

The emissary executor determines the command to run in this order:

1. Command specified in the workflow spec
2. Command from the image index cache
3. Command from the container image

The image index is a [configuration item](workflow-controller-configmap.yaml), called `images`.

The controller creates a cache entry using the image with version as key and command as value.
It reuses this cache for specific image:version combinations, so you may get surprising behavior if you update the command in an image without changing its version tag.

### Troubleshooting

The emissary will exit with code 64 if it fails.
This may indicate a bug in the emissary.
