Description: Experimental fast-path reconciliation for newly terminal taskset results
Authors: [droctothorpe](https://github.com/droctothorpe)
Component: General
Issues: 17228

Prototype: disabled by default; enable `ARGO_TASKSET_COMPLETION_FAST_PATH=true` on the controller to enqueue newly terminal taskset results immediately.
Routine updates and retries for stale workflow versions retain the existing delay.
Agent result batching and pod-completion handling are unchanged.
This prototype still needs cluster-scale latency and controller-load validation.
