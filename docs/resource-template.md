# Resource Template

> v2.0 and after

See [Kubernetes Resources](walk-through/kubernetes-resources.md).

## Agent-based execution

When resource templates are executed by the agent instead of a per-node pod, be aware of the following:

* Agent-based resource templates run in their own pod, named `<workflow>-<id>-resource-agent` and
  labeled `workflows.argoproj.io/component: resource-agent`, separate from the HTTP/plugin agent
  pod. Like the HTTP agent, one such pod serves all resource-template nodes of the workflow.
* Created resources are labeled with `workflows.argoproj.io/agent-resource: <workflow UID>` and
  annotated with `workflows.argoproj.io/node-id` plus the template's success/failure conditions.
  The agent's informers select on the label and evaluate the annotations, so these must not be
  stripped by admission controllers or other mutating controllers.
* Node results are reported at most once per node. If the agent pod restarts, an
  already-completed node's result may be patched into the `WorkflowTaskSet` a second time; this
  is harmless.
* Output parameters (`jsonPath`/`jqFilter`) are resolved from the watched object's state at the
  moment its success conditions are met, not via `kubectl get`.

### Service account and RBAC

The resource agent watches every resource kind your templates create, which requires `list` and
`watch` on the whole kind — a broader grant than workflow pods should carry. It therefore runs
under a dedicated service account named `<workflow service account>-resource-agent` (for example
`default-resource-agent` for workflows using the `default` service account). The workflow errors
if this service account does not exist.

The service account needs:

* `list` and `watch` on `workflowtasksets`, and `patch` on `workflowtasksets/status` (to receive
  tasks and report results, the same as the [HTTP agent](workflow-rbac.md)),
* `create`, `list` and `watch` on every resource kind your templates create,
* `get` on `secrets` and `configmaps` if you use `manifestFrom` artifacts, since the agent
  resolves them via the Kubernetes API rather than an init container.

For example, for templates that create `sparkapplications`:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: resource-agent
rules:
  - apiGroups:
      - argoproj.io
    resources:
      - workflowtasksets
    verbs:
      - list
      - watch
  - apiGroups:
      - argoproj.io
    resources:
      - workflowtasksets/status
    verbs:
      - patch
  - apiGroups:
      - sparkoperator.k8s.io
    resources:
      - sparkapplications
    verbs:
      - create
      - list
      - watch
```
