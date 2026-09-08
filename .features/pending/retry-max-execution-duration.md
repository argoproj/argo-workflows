Description: Bound retries by cumulative Pod execution time
Authors: [Herik Webb](https://github.com/herikwebb)
Component: General
Issues: 16857

`retryStrategy.maxExecutionDuration` adds an optional cumulative execution-time budget for completed Pod-backed attempts that fail or error.
Argo counts each attempt as one wall-clock envelope from the earliest observed main-container start through the latest main-container finish, including gaps within a ContainerSet but never double-counting overlapping containers.
If the available timestamps do not establish the finish, Argo conservatively counts from the earliest observed main-container start until it observes the attempt complete.
Pod scheduling and Pending time, image pulling, init-container time, post-main output processing, and retry back-off waits do not consume this budget.
Attempts that fail before any main container starts do not consume this budget.
Argo checks the cumulative total only after an attempt fails or errors and before starting another retry, so it never terminates an active attempt and accepts a successful attempt even if its execution reaches or exceeds the budget.
Argo supports this setting on `container`, `script`, `containerSet`, `resource`, and `data` templates and captures a resolved parameterized value when the retry sequence starts.
`maxExecutionDuration` is independent of `backoff.maxDuration`, so users can configure both execution-time and wall-clock bounds and whichever prevents the next retry first takes effect.
Users can also combine it with `limit` or `pendingTimeout` when attempts that never start must remain bounded.
This feature also hardens automatic failed-Pod restart classification: if Kubernetes preserves a terminated main-container run in `lastState`, Argo no longer treats that Pod as a pre-start infrastructure failure.
