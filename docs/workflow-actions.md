# Workflow Actions

> v4.2 and after

A `WorkflowAction` requests a lifecycle action on a Workflow: `Suspend`, `Resume`, `Stop` or `Terminate`.
Instead of clients modifying the Workflow directly, the workflow controller performs the action inside its reconciliation loop and reports the outcome on the WorkflowAction's `status`.
This makes actions serialized with reconciliation, observable when they fail, and available to `kubectl`-only users without the Argo Server.

The Argo Server's stop, terminate, suspend and resume endpoints, the CLI, and the UI all use WorkflowActions automatically.
You only need to create WorkflowActions yourself when working directly against the Kubernetes API.

## Example

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowAction
metadata:
  generateName: my-wf-resume-
spec:
  workflowRef:
    name: my-wf
  action: Resume
  resume:
    nodeFieldSelector: displayName=approve
    outputParameters:
      approved: "true"
```

Create it in the same namespace as the target Workflow:

```bash
kubectl create -f action.yaml
kubectl get wfa
```

The `resume` parameter block is only valid with `action: Resume`, and `stop` only with `action: Stop`:

```yaml
spec:
  workflowRef:
    name: my-wf
  action: Stop
  stop:
    message: stopped by the release manager
    nodeFieldSelector: displayName=approve  # optional: fail matching suspended nodes instead of stopping the whole workflow
```

`Suspend` and `Terminate` take no parameters.
The spec is immutable after creation.

## Outcome

The controller records the outcome on the action's status:

```yaml
status:
  phase: Succeeded
  completionTime: "2026-09-09T12:00:00Z"
```

| Phase | Reason | Meaning |
|---|---|---|
| `Pending` (or empty) | | The controller has not yet processed the action. |
| `Succeeded` | | The action was applied, or was already in effect (actions are idempotent). |
| `Failed` | `WorkflowNotFound` | The target Workflow does not exist. The action is not retried. |
| `Failed` | `WorkflowCompleted` | The target Workflow already completed. |
| `Failed` | `InvalidAction` | The action could not be applied, for example a `nodeFieldSelector` matching no suspended nodes; `message` has the details. |

A Stop or Terminate accepted by the controller is recorded in the Workflow's `status.shutdown`, which supersedes `spec.shutdown`.
The Workflow's `status.suspended` records the suspension state accepted by the controller, mirrored with the deprecated `spec.suspend` so older clients can still suspend and resume; Resume clears both.
The requester's identity labels on the action (set by the Argo Server) are copied to the Workflow when the action is applied.

## Garbage collection

The controller deletes actions after they reach a terminal phase, once the TTL configured by `workflowActionTTL` in the [workflow controller ConfigMap](workflow-controller-configmap.md) expires.
The default is 24 hours.

## RBAC

Requesting an action needs only `create` on `workflowactions` — strictly less than the `patch` on `workflows` the old mechanism required.
The bundled aggregate cluster roles grant `workflowactions` access alongside the other Argo resources.

To allow some verbs but not others, use a
[ValidatingAdmissionPolicy](https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/) — no webhook or policy engine required:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: deny-terminate-for-devs
spec:
  matchConstraints:
    resourceRules:
      - apiGroups: ["argoproj.io"]
        apiVersions: ["v1alpha1"]
        operations: ["CREATE"]
        resources: ["workflowactions"]
  validations:
    - expression: >-
        object.spec.action != 'Terminate' ||
        !('developers' in request.userInfo.groups)
      message: developers may not terminate workflows
---
# a policy does nothing until a binding enforces it
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicyBinding
metadata:
  name: deny-terminate-for-devs
spec:
  policyName: deny-terminate-for-devs
  validationActions: [Deny]
```

## Compatibility

Directly setting `spec.shutdown` or `spec.suspend` on a Workflow still works in this release, and is deprecated for removal in a later release; the [`deprecated_feature`](metrics.md#deprecated_feature) metric counts remaining uses.
To create a Workflow that starts in the suspended state, set `spec.startSuspended: true` instead of `spec.suspend`.
An Argo Server from this release requires a workflow controller from this release to perform actions; against an older controller the action endpoints time out after 30 seconds.
