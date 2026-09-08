Description: Count retries suppressed by retry strategy duration budgets
Authors: [Herik Webb](https://github.com/herikwebb)
Component: Telemetry
Issues: 16856

The controller now exposes `retry_strategy_terminations_total` when an otherwise eligible retry is suppressed because `maxDuration` has elapsed or its next back-off would exceed the deadline.
The bounded `reason` attribute distinguishes `MaxDurationExceeded` from `BackoffWouldExceedMaxDuration`, and the `namespace` attribute identifies the Workflow namespace.
`retry_strategy_terminations_total` is now a reserved controller metric name, so rename any existing custom metric with that name before upgrading.
