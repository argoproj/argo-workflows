# Shared Agent Pod

The shared agent pod mode lets multiple workflows reuse one agent pod per service account.
Use this mode when many workflows in the same namespace run HTTP or plugin templates.
This reduces pod churn and keeps service account isolation.

In default mode, each workflow creates its own agent pod.
In shared mode, the controller creates `argo-agent-{serviceAccountName}` and reuses it.

## Prerequisites

- Executor plugins are enabled on the workflow controller as described in [Executor Plugins](executor_plugins.md#configuration).
- Agent RBAC is configured as described in [Argo Agent RBAC](http-template.md#argo-agent-rbac).
- You can update `workflow-controller-configmap` in the controller namespace.

!!! Note
    Shared mode is configured in the controller ConfigMap and applies to workflows handled by that controller.

## Setup

1. Edit `workflow-controller-configmap` and add or update the `agent` section.

    ```yaml
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: workflow-controller-configmap
      namespace: argo
    data:
      agent: |
        runMultipleWorkflow: true
        deleteAfterCompletion: false
        createPod: true
        resources:
          requests:
            cpu: 10m
            memory: 64Mi
          limits:
            cpu: 100m
            memory: 256Mi
        securityContext:
          runAsNonRoot: true
          runAsUser: 8737
          capabilities:
            drop:
              - ALL
    ```

2. Default configuration behavior:

    - `runMultipleWorkflow` default is `false`.
    - `deleteAfterCompletion` default is `true`.
    - `createPod` default is `true`.

!!! Warning
    If `deleteAfterCompletion: false`, shared agent pods remain running after workflows complete.
    This can increase baseline cluster resource usage.

### Agent configuration keys

- `runMultipleWorkflow`: Enables shared mode when set to `true`.
  In shared mode, one agent pod is reused per service account.
  When set to `false`, each workflow uses its own agent pod.
- `deleteAfterCompletion`: Controls shared agent pod cleanup.
  When set to `true`, the shared agent pod is deleted after completion checks pass.
  When set to `false`, the shared agent pod stays running for reuse.
- `createPod`: Controls whether the workflow controller creates agent pods.
  When set to `true`, the controller creates and manages agent pods.
  When set to `false`, pod creation is external and the controller only reconciles `WorkflowTaskSet` resources.

**Tip:** Start with `deleteAfterCompletion: true` in production and change it only if startup churn is a problem.

**Tip:** Use `createPod: false` if you want `WorkflowTaskSet` resources to be updated by an external operator or service. This removes the workflow controller's role in creating the agent pod. This is useful for cluster-level operators or multi-tenant operators.

## Example

This example submits two workflows that use HTTP templates with the same service account.
Both workflows should reuse one shared agent pod.

1. Create an example workflow manifest.

    ```bash
    cat <<'EOF' >/tmp/shared-agent-http.yaml
    apiVersion: argoproj.io/v1alpha1
    kind: Workflow
    metadata:
      generateName: shared-agent-http-
    spec:
      serviceAccountName: default
      entrypoint: main
      templates:
        - name: main
          steps:
            - - name: call-http
                template: http
        - name: http
          http:
            url: https://httpbin.org/get
            method: GET
            timeoutSeconds: 30
    EOF
    ```

2. Submit the workflow twice.

    ```bash
    kubectl -n argo create -f /tmp/shared-agent-http.yaml
    kubectl -n argo create -f /tmp/shared-agent-http.yaml
    ```

3. Watch both workflows.

    ```bash
    kubectl -n argo get wf -w
    ```

## Verification

Check that one shared agent pod exists for the `default` service account.

```bash
kubectl -n argo get pods -l workflows.argoproj.io/component=agent
kubectl -n argo get pods -l workflows.argoproj.io/agent-service-account=default
```

Expected output includes one pod named `argo-agent-default`.

## Related links

- [Executor Plugins](executor_plugins.md)
- [HTTP Template](http-template.md)
- [Argo Agent RBAC](http-template.md#argo-agent-rbac)
- [Workflow Controller ConfigMap](workflow-controller-configmap.md#agentconfig)
- [Environment Variables](environment-variables.md)
