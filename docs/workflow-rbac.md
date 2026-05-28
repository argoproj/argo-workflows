# Workflow RBAC

All pods in a workflow run with the service account specified in `workflow.spec.serviceAccountName`, or if omitted,
the `default` service account of the workflow's namespace. The amount of access which a workflow needs is dependent on
what the workflow needs to do. For example, if your workflow needs to deploy a resource, then the workflow's service
account will require 'create' privileges on that resource.

**Warning**: We do not recommend using the `default` service account in production. It is a shared account so may have
permissions added to it you do not want. Instead, create a service account only for your workflow.

The minimum for the executor to function:

For >= v3.4:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: executor
rules:
  - apiGroups:
      - argoproj.io
    resources:
      - workflowtaskresults
    verbs:
      - create
      - patch
```

## Agent RBAC for resource templates

Resource templates (`resource:` step type) are executed by the workflow's agent pod, not by a per-step wait container. The agent watches each created object via a dynamic informer (filtered by a `workflows.argoproj.io/monitored-resource=<workflowName>` label) to evaluate `successCondition` / `failureCondition`.

This means the agent's `ServiceAccount` needs `list` and `watch` on **every GVR a resource template creates**, on top of any `create`/`delete`/`patch` verbs the action itself requires. Without these the informer's cache sync fails and the node is marked failed with a `watch <gvr>: …` message.

Example role for a workflow that uses a resource template to create child `Workflows` (workflow-of-workflows) and `Jobs`:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: agent
rules:
  # Required for every agent-using workflow:
  - apiGroups: ["argoproj.io"]
    resources: ["workflowtasksets"]
    verbs: ["list", "watch"]
  - apiGroups: ["argoproj.io"]
    resources: ["workflowtasksets/status"]
    verbs: ["patch"]

  # Per resource-template GVR — create/delete for the action, list/watch for the informer:
  - apiGroups: ["argoproj.io"]
    resources: ["workflows"]
    verbs: ["create", "delete", "list", "watch"]
  - apiGroups: ["batch"]
    resources: ["jobs"]
    verbs: ["create", "delete", "list", "watch"]
```

If you use a `delete` action without success/failure conditions, only `delete` is required — the agent short-circuits before starting an informer. For every other action you need `list` and `watch`.
