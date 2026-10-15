Description: Experimental separate reconciliation delay for newly terminal taskset results
Authors: [droctothorpe](https://github.com/droctothorpe)
Component: General
Issues: 17228

Prototype: leave `ARGO_TASKSET_COMPLETION_REQUEUE_TIME` unset to preserve the current rate-limited behavior.
Set it on the controller to a non-negative duration such as `250ms` or `1s` to independently schedule newly terminal taskset results; `0s` enqueues them immediately.
Empty, negative and malformed values are rejected at startup.
Routine updates and retries for stale workflow versions retain the existing delay.
This is not a minimum reconciliation interval: other events may enqueue a workflow sooner.
Agent result batching and pod-completion handling are unchanged.
This prototype still needs cluster-scale latency and controller-load validation.
