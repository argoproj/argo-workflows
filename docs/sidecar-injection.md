# Sidecar Injection

Automatic (i.e. mutating webhook based) sidecar injection systems, including service meshes such as Anthos and Istio
Proxy, create a unique problem for Kubernetes workloads that complete.

Because sidecars are injected outside of view of the workflow controller, the controller has no awareness of them. It has no opportunity to rewrite the containers
command (when using the Emissary Executor) and as the sidecar's process will run as PID 1, which is protected, it may not be possible for the the wait container to terminate the sidecar.

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