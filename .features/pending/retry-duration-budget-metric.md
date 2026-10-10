Description: Count retries suppressed by a `retryStrategy` duration budget
Authors: [Abhishek Sharma](https://github.com/abhisheksharma2411)
Component: Telemetry
Issues: 16856

A `retryStrategy` duration budget can stop a retry that would otherwise have gone ahead.
That happens when `backoff.maxDuration` has already elapsed, or when waiting out the next back-off would cross it.
Until now that decision appeared only in the retry node's message, which explains one Workflow but gives operators nothing to alert on.

The controller now emits a `retry_strategy_terminations_total` counter.
Each increment is one attempt the controller would have made and did not, because of the duration budget.
The counter carries only `namespace` and a `reason` of `MaxDurationExceeded` or `BackoffWouldExceedMaxDuration`, so its cardinality does not grow with the number of Workflows, nodes or templates.

The counter does not increment when the retry would have been rejected anyway by `retryPolicy`, `limit` or `expression`.
Attributing those to the duration budget would make the metric answer which check the controller ran first rather than why the retry stopped.

The retry node's message is unchanged and remains the signal for diagnosing an individual Workflow.
