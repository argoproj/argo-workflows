Description: Prototype minimum delay for completion-triggered reconciliation
Authors: [droctothorpe](https://github.com/droctothorpe)
Component: General
Issues: 17228

This opt-in prototype sets the minimum delay before reconciling a workflow after a completion, currently applying only to HTTP and executor-plugin nodes whose results arrive through a WorkflowTaskSet.
Set `COMPLETED_REQUEUE_TIME_MS` on the controller to a positive integer number of milliseconds, for example `250` or `1000`; zero, negative, malformed and overflowing values fail startup.
`DEFAULT_REQUEUE_TIME` caps the completion delay.
The agent still batches results on its own ticker, so lower `ARGO_AGENT_PATCH_RATE` too to see much benefit.
There is no per-workflow throttling, so a large executor-plugin fan-out can reconcile at roughly the configured rate, increasing load on the controller, API server and etcd.
Gains may vary under load: if the workflow informer is behind the controller's last write, the reconcile is rejected as outdated and waits the full `DEFAULT_REQUEUE_TIME` before retrying.
