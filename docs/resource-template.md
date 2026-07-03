# Resource Template

> v2.0 and after

See [Kubernetes Resources](walk-through/kubernetes-resources.md).

## Agent-based execution

When resource templates are executed by the agent instead of a per-node pod, be aware of the following:

* Created resources are labeled with `workflows.argoproj.io/agent-resource: <workflow UID>` and
  annotated with `workflows.argoproj.io/node-id` plus the template's success/failure conditions.
  The agent's informers select on the label and evaluate the annotations, so these must not be
  stripped by admission controllers or other mutating controllers.
* The agent's service account needs `list` and `watch` on every resource kind your templates
  create (to watch conditions), and `get` on `secrets` and `configmaps` if you use
  `manifestFrom` artifacts, since the agent downloads them via the Kubernetes API rather than an
  init container.
* Node results are reported at most once per node. If the agent pod restarts, an
  already-completed node's result may be patched into the `WorkflowTaskSet` a second time; this
  is harmless.
* Output parameters (`jsonPath`/`jqFilter`) are resolved from the watched object's state at the
  moment its success conditions are met, not via `kubectl get`.
