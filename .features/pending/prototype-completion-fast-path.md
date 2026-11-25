Description: Prototype minimum delay for completion-triggered reconciliation
Authors: [droctothorpe](https://github.com/droctothorpe), [ricCap](https://github.com/ricCap)
Component: General
Issues: 17228, 17244

This opt-in prototype sets the minimum delay before reconciling a workflow after a node completes.
It applies to HTTP and executor-plugin nodes whose results arrive through a WorkflowTaskSet, and to pod nodes.
A pod node completes only when its pod has finished and its WorkflowTaskResult has reported its outputs. These arrive as separate events, and only the later one uses the shorter delay.
Set `COMPLETED_REQUEUE_TIME_MS` on the controller to a positive integer number of milliseconds, for example `250` or `1000`; zero, negative, malformed and overflowing values fail startup.
`DEFAULT_REQUEUE_TIME` caps the completion delay.
For HTTP and executor-plugin nodes, the agent still batches results on its own ticker, so lower `ARGO_AGENT_PATCH_RATE` too to see much benefit.
The controller increases completion delays for expensive workflows using `COMPLETED_REQUEUE_DUTY_CYCLE_PERCENT`, which defaults to `20` percent of one worker's time per workflow.
Gains may vary under load: if the workflow informer is behind the controller's last write, the reconcile is rejected as outdated and waits the full `DEFAULT_REQUEUE_TIME` before retrying.
