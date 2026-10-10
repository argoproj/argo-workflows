Description: Hashed `H` schedules to spread CronWorkflows out over time
Authors: [krrrr38](https://github.com/krrrr38)
Component: CronWorkflows
Issues: 16603

`spec.schedules` now accepts `H` (hash), the same way Jenkins does, for example `H H * * *` to run once a day at a fixed but arbitrary time.
A `H` resolves to a value derived from the `CronWorkflow`'s namespace and name, so it never changes for a given `CronWorkflow`.
This avoids the load spike you get when many `CronWorkflows` share a round schedule such as `0 0 * * *` and all fire at the same instant, which is common when the schedule comes from a chart or a generator.
The schedules you configure are never rewritten: the new `status.resolvedSchedules` reports what each `H` resolved to, and `argo cron get` and the UI show it alongside the configured schedule.
See [hashed schedules](cron-workflows.md#hashed-schedules) for the full syntax.
