Description: Metrics to observe mutex and semaphore locks, so users can detect unreleased locks that are blocking workflows
Authors: [Jason Meridth](https://github.com/jmeridth) [Alan Clucas](https://github.com/Joibel)
Component: Telemetry
Issues: 14888

The controller now emits telemetry for synchronization locks (mutexes and semaphores), letting you detect unreleased locks that are causing workflow blocks or timeouts.

Three metrics are exposed, each labelled by `type` (`mutex` or `semaphore`), `storage` (`configmap` or `database`), `lock_name` and `namespace`:
  - `locks_taken_total` — a counter of how many locks have been acquired, for throughput and churn (`rate()`).
  - `locks_held` — a gauge of how many holders currently hold each lock right now.
  - `locks_pending` — a gauge of how many workflows are currently waiting to acquire each lock.

For database-backed locks (which are shared across controllers and clusters) each controller reports only its own contribution, using a single per-controller aggregate query per scrape.
