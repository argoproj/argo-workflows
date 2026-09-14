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
    uid: 9c1f4b2e-...  # optional: fail rather than act on a Workflow recreated under the same name
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
kubectl describe wfa my-wf-resume-abcde  # the outcome is also recorded as an Event on the action
```

`outputParameters` requires a `nodeFieldSelector`; the API rejects an action that sets one without the other.
If the controller is configured with an [instance ID](scaling.md#declarative-usage), the action must carry the `workflows.argoproj.io/controller-instanceid` label, like every other resource that controller watches; without it the action is never processed.

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
| `Failed` | `WorkflowNotFound` | The target Workflow does not exist, or exists with a different `uid` than the one in `workflowRef`. The action is not retried. |
| `Failed` | `WorkflowCompleted` | The target Workflow already completed. |
| `Failed` | `InvalidAction` | The action could not be applied, for example a `nodeFieldSelector` matching no suspended nodes; `message` has the details. |

A Stop or Terminate accepted by the controller is recorded in the Workflow's `status.shutdown`, which supersedes `spec.shutdown`.
The Workflow's `status.suspended` records the suspension state accepted by the controller, mirrored with the deprecated `spec.suspend` so older clients can still suspend and resume; Resume clears both.
The requester's identity labels on the action (set by the Argo Server) are copied to the Workflow when the action is applied, replacing those of any earlier action.
The controller emits an Event on the action and on the Workflow when it records the outcome.
Pending actions are applied in `creationTimestamp` order; two actions created within the same second have no defined order relative to each other.

## Garbage collection

The controller deletes actions after they reach a terminal phase, once the TTL configured by `workflowActionTTL` in the [workflow controller ConfigMap](workflow-controller-configmap.md) expires.
The default is 24 hours.

## RBAC

Creating an action with `kubectl` needs only `create` on `workflowactions`.
Through the Argo Server the request is made with the caller's own identity (in `client` and `sso` [auth modes](argo-server-auth-mode.md)), and the server also `get`s and `watch`es the action to report its outcome, so the caller needs `create`, `get` and `watch` on `workflowactions`.
Neither needs the `patch` on `workflows` that the old mechanism required.
The bundled aggregate cluster roles grant `workflowactions` access alongside the other Argo resources; hand-written roles must be extended when upgrading (see the [upgrading guide](upgrading.md)).

To allow some actions but not others, use a [ValidatingAdmissionPolicy](https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/) — no webhook or policy engine required:

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

Directly setting `spec.shutdown` or `spec.suspend` on a running Workflow still works in this release, and is deprecated for removal in a later release; the [`deprecated_feature`](metrics.md#deprecated_feature) metric counts remaining uses.
Setting `spec.suspend` at creation time ("start suspended") remains supported; `spec.startSuspended: true` does the same and is preferred.
An Argo Server from this release requires a workflow controller from this release to perform actions; against an older controller the action endpoints time out after 30 seconds.
This applies to the `argo` CLI without an Argo Server too (the Kubernetes API mode): it creates a WorkflowAction and waits for the controller, so the controller must be running, and the `WorkflowAction` CRD installed, for `argo stop`, `argo terminate`, `argo suspend` and `argo resume` to work.
When the endpoints do time out, the action has still been recorded and is applied once the controller gets to it; `kubectl get wfa` shows its outcome.
