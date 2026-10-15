Description: Independent completion delay for `WorkflowTaskSet` results
Authors: [droctothorpe](https://github.com/droctothorpe)
Component: General
Issues: 17228

Short HTTP or executor-plugin steps can spend longer waiting for reconciliation than running.
Set `ARGO_TASKSET_COMPLETION_REQUEUE_TIME` on the controller to a non-negative duration, for example `250ms` or `1s`, to independently schedule newly terminal `WorkflowTaskSet` results; `0s` enqueues them immediately once the controller receives them.
Leave the setting unset to retain normal rate limiting.
To also reduce the wait before results reach the controller, lower `ARGO_AGENT_PATCH_RATE`, which defaults to `DEFAULT_REQUEUE_TIME`.
