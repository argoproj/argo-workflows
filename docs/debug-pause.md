# Debug Pause

> v3.3 and after

## Introduction

The `debug pause` feature makes it possible to pause individual workflow steps for debugging before, after or both and then release the steps from the paused state. Currently this feature is only supported when using the [Emissary Executor](workflow-executors.md#emissary-emissary)

In order to pause a container env variables are used:

- `ARGO_DEBUG_PAUSE_AFTER` - to pause a step after execution
- `ARGO_DEBUG_PAUSE_BEFORE` - to pause a step before execution

Example workflow:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: pause-after-
spec:
  entrypoint: argosay
  templates:
    - name: argosay
      container:
        image: argoproj/argosay:v2
        env:
          - name: ARGO_DEBUG_PAUSE_AFTER
            value: 'true'
```

In order to release a step from a pause state, marker files are used named `/var/run/argo/ctr/main/after` or `/var/run/argo/ctr/main/before` corresponding to when the step is paused. Pausing steps can be used together with [ephemeral containers](https://kubernetes.io/docs/concepts/workloads/pods/ephemeral-containers/) when a shell is not available in the used container.

## Example

1) Create a workflow where the debug pause env in set, in this example `ARGO_DEBUG_PAUSE_AFTER` will be set and thus the step will be paused after execution of the user code.

pause-after.yaml

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: pause-after-
spec:
  entrypoint: argosay
  templates:
    - name: argosay
      container:
        image: argoproj/argosay:v2
        env:
          - name: ARGO_DEBUG_PAUSE_AFTER
            value: 'true'
```

```bash
argo submit -n argo --watch pause-after.yaml
```

Create a shell in the container of interest of create a ephemeral container in the pod, in this example ephemeral containers are used.

```bash
kubectl debug -n argo -it POD_NAME --image=busybox --target=main --share-processes
```

In order to have access to the persistence volume used by the workflow step,  [`--share-processes`](https://kubernetes.io/docs/tasks/configure-pod-container/share-process-namespace/) will have to be used.

The ephemeral container can be used to perform debugging operations. When debugging has been completed, create the marker file to allow the workflow step to continue. When using process name space sharing container file systems are visible to other containers in the pod through the `/proc/$pid/root` link.

```bash
touch /proc/1/root/var/run/argo/ctr/main/after
```
