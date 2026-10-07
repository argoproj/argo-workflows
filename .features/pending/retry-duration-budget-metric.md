Description: Count retries suppressed by a `retryStrategy` duration budget
Authors: [Abhishek Sharma](https://github.com/abhisheksharma2411)
Component: Controller
Issues: 16856

A `retryStrategy` duration budget can stop a retry that would otherwise have gone ahead, either because `backoff.maxDuration` has already elapsed or because waiting out the next back-off would cross it.
Until now that decision was visible only in the retry node's message, which answers "why did this Workflow stop" but gives operators no aggregate signal to alert on.

The controller now emits a `retry_strategy_terminations_total` counter.
Each increment is one attempt the controller would have made and did not, because of the duration budget.
The counter carries only `namespace` and a `reason` of `MaxDurationExceeded` or `BackoffWouldExceedMaxDuration`, so its cardinality does not grow with the number of Workflows, nodes or templates.

The counter deliberately does not increment when the retry would have been rejected anyway by its retry policy, the node's own retryability, `limit` or `expression`.
Those checks run after the duration checks, so a retry blocked by `limit` would otherwise be reported as a duration-budget termination purely because the duration check happens to be evaluated first.
Attributing it that way would make the metric answer "which check ran first" rather than "why did the retry stop".

The retry node's message is unchanged and remains the durable signal for diagnosing an individual Workflow.
