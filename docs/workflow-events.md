# Workflow Events

> v2.7.2

⚠️ Do not use Kubernetes events for automation. Events maybe lost or rolled-up.

We emit Kubernetes events on certain events.

Workflow state change:

* `WorkflowRunning`
* `WorkflowSucceeded`
* `WorkflowFailed`
* `WorkflowTimedOut`

Node state change:

* `WorkflowNodeRunning`
* `WorkflowNodeSucceeded`
* `WorkflowNodeFailed`
* `WorkflowNodeError`

The involved object is the workflow in both cases. Additionally, for node state change events, annotations indicate the name and type of the involved node:

```yaml
metadata:
  name: my-wf.160434cb3af841f8
  namespace: my-ns
  annotations:
    workflows.argoproj.io/node-name: my-node
    workflows.argoproj.io/node-type: Pod
type: Normal
reason: WorkflowNodeSucceeded
message: 'Succeeded node my-node: my message'
involvedObject:
  apiVersion: v1alpha1
  kind: Workflow
  name: my-wf
  namespace: my-ns
  resourceVersion: "1234"
  uid: my-uid
firstTimestamp: "2020-04-09T16:50:16Z"
lastTimestamp: "2020-04-09T16:50:16Z"
count: 1
```
