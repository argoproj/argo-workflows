# Sidecar Injection

Automatic (i.e. mutating webhook based) sidecar injection systems, including service meshes such as Anthos and Istio
Proxy, create a unique problem for Kubernetes workloads that run to completion.

Because sidecars are injected outside of the view of the workflow controller, the controller has no awareness of them.
It has no opportunity to rewrite the containers command (when using the Emissary Executor) and as the sidecar's process
will run as PID 1, which is protected. It can be impossible for the wait container to terminate the sidecar.

You will minimize problems by not using Istio with Argo Workflows.

See [#1282](https://github.com/argoproj/argo-workflows/issues/1282).

## Support Matrix

Key:

* Any - we can kill any image
* KubectlExec - we kill images by running `kubectl exec`

| Executor | Sidecar | Injected Sidecar |
|---|---|---|
| `emissary` | Any | KubectlExec |

## How We Kill Sidecars Using `kubectl exec`

> v3.1 and after

Kubernetes does not provide a way to kill a single container. You can delete a pod, but this kills all containers, and loses all information
and logs of that pod.

Instead, try to mimic the Kubernetes termination behavior, which is:

1. SIGTERM PID 1
1. Wait for the pod's `terminateGracePeriodSeconds` (30s by default).
1. SIGKILL PID 1

The following are not supported:

* `preStop`
* `STOPSIGNAL`

To do this, it must be possible to run a `kubectl exec` command that kills the injected sidecar. By default it runs `/bin/sh -c 'kill 1'`. This can fail:

1. No `/bin/sh`.
2. Process is not running as PID 1 (which is becoming the default these days due to `runAsNonRoot`).
3. Process does not correctly respond to `kill 1` (e.g. some shell script weirdness).

You can override the kill command by using a pod annotation (where `%d` is the signal number), for example:

```yaml
spec:
  podMetadata:
    annotations:
      workflows.argoproj.io/kill-cmd-istio-proxy: '["pilot-agent", "request", "POST", "quitquitquit"]'
      workflows.argoproj.io/kill-cmd-vault-agent: '["sh", "-c", "kill -%d 1"]'
      workflows.argoproj.io/kill-cmd-sidecar: '["sh", "-c", "kill -%d $(pidof entrypoint.sh)"]'
```
