Description: WorkflowAction CRD: stop, terminate, suspend and resume are performed by the workflow controller
Authors: [Alan Clucas](https://github.com/Joibel)
Component: General
Issues: 16741 2942 12538 12863

Lifecycle actions on a Workflow can be requested by creating a WorkflowAction resource, with `kubectl` or through the Argo Server.
The workflow controller performs the action inside its reconciliation loop and reports the outcome on the WorkflowAction's `status`, so a lost or invalid request is visible instead of silently ignored.
The Argo Server's stop, terminate, suspend and resume endpoints now create WorkflowActions and wait up to 30 seconds for the controller to act, preserving their existing request and response shapes.
Stop and terminate intent accepted by the controller is recorded in the Workflow's `status.shutdown`, which supersedes `spec.shutdown`.
Directly setting `spec.shutdown` or `spec.suspend` on a running Workflow still works in this release, and is deprecated for removal in a later release.
An Argo Server from this release requires a workflow controller from this release to perform these actions; against an older controller the endpoints time out.
Completed WorkflowActions are deleted by the controller after a configurable TTL (`workflowActionTTL`, default 24h).
