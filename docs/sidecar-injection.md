# Sidecar Injection

Automatic (that is, mutating webhook based) sidecar injection systems, including service meshes such as Anthos and Istio Proxy, create a unique problem for Kubernetes workloads that run to completion.

Because the webhook injects sidecars outside of the view of the workflow controller, the controller has no awareness of them.
The controller has no opportunity to rewrite the container's command.
The sidecar's process runs as PID 1, which can prevent signals from terminating it.
This can make it impossible for the workflow controller to terminate the sidecar.

You can minimize problems by not using Istio with Argo Workflows.

See [#1282](https://github.com/argoproj/argo-workflows/issues/1282).

## Support Matrix

Key:

* Any - Argo can kill any image
* KubectlExec - Argo kills images by running `kubectl exec`

| Executor | Sidecar | Injected Sidecar |
|---|---|---|
| `emissary` | Any | KubectlExec |

## How Argo Kills Sidecars Using `kubectl exec`

Kubernetes does not provide a way to kill a single container.
You can delete a pod, but this kills all containers, and loses all information and logs of that pod.

Instead, Argo mimics the Kubernetes termination behavior, which is:

1. SIGTERM PID 1
1. Wait for the pod's `terminationGracePeriodSeconds` (30s by default).
1. SIGKILL PID 1

Argo doesn't support the following:

* `preStop`
* `STOPSIGNAL`

To do this, Argo runs a `kubectl exec` command that kills the injected sidecar.
By default it runs `/bin/sh -c 'kill -<signal> 1'`.
This can fail:

1. No `/bin/sh`.
2. Process is not running as PID 1.
3. Process does not correctly respond to the signal (for example, some shell script weirdness).

You can override the kill command by using a pod annotation (where `%d` is the signal number), for example:

```yaml
spec:
  podMetadata:
    annotations:
      workflows.argoproj.io/kill-cmd-istio-proxy: '["pilot-agent", "request", "POST", "quitquitquit"]'
      workflows.argoproj.io/kill-cmd-linkerd-proxy: '["/usr/lib/linkerd/linkerd-await","sleep","1","--shutdown"]'
      workflows.argoproj.io/kill-cmd-vault-agent: '["sh", "-c", "kill -%d 1"]'
      workflows.argoproj.io/kill-cmd-sidecar: '["sh", "-c", "kill -%d $(pidof entrypoint.sh)"]'
```
