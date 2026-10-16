Description: Per-workflow duty cycle for completion-triggered reconciliation
Authors: [droctothorpe](https://github.com/droctothorpe)
Component: General
Issues: 17244

When you enable `COMPLETED_REQUEUE_TIME_MS`, the controller uses each workflow's smoothed reconciliation duration to increase completion delays for expensive workflows.
The controller collects duty-cycle timing and utilization samples only while this setting is enabled.
Set `COMPLETED_REQUEUE_DUTY_CYCLE_PERCENT` to an integer from `1` to `100` to target a percentage of one worker's time per workflow; the default is `20`, and `100` adds no gap beyond the completion floor.
The delay stays between the configured completion floor and `DEFAULT_REQUEUE_TIME`, with the ceiling taking precedence if the floor is higher.
The `workflow_completion_requeue_delay` and `workflow_reconcile_utilization` histograms report requested delays and observed per-workflow worker utilization without workflow-name or UID labels.
