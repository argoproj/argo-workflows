# Sidecar Injection

Automatic (i.e. mutating webhook based) sidecar injection systems, including service meshes such as Anthos and Istio
Proxy, create a unique problem for Kubernetes workloads that complete.

Because sidecars are injected outside of view of the workflow controller, the controller has no awareness of them. It
has no opportunity to rewrite the containers command (when using the Emissary Executor) and as the sidecar's process
will run as PID 1, which is protected, it may not be possible for the the wait container to terminate the sidecar.

You will minimize problems by not using Istio with Argo Workflows.

You can disable Istio's automatic injection for workflow pods:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: istio-sidecar-injector
data:
  config: |-
    neverInjectSelector:
      - matchExpressions:
        - key: workflows.argoproj.io/workflow
          operator: Exists
```

See [#1282](https://github.com/argoproj/argo-workflows/issues/1282).

## How We Kill Sidecars

Kubernetes does provide anyway to kill a single container, aside from deleting a pod, and thereby loosing all
information and logs of that pod.

Instead, we want mimic the standard termination behaviour as follows:

1. SIGTERM all the processing in the container `kubectl exec -ti ${POD_NAME} -c ${SIDECAR_NAME} -- kill -15 1`.
1. Wait for the pod's `terminateGracePeriodSeconds` (30s by default).
1. SIGKILL all the processing in the container `kubectl exec -ti ${POD_NAME} -c ${SIDECAR_NAME} -- kill -9 1`

The following are not supported:

* `preStop`
* `STOPSIGNAL`

\| Executor | Sidecar | Injected Sidecar | |---|---|---| | docker | All | All | | emissary | All | Debian | | k8sapi |
Debian | Debian | | kubelet | Debian | Debian | | pns | All | All | 