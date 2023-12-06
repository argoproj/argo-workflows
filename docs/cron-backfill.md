# Cron Backfill

## Use Case

* You are using cron workflows to run daily jobs, you may need to re-run for a date, or run some historical days.

## Solution

1. Create a workflow template for your daily job.
2. Create your cron workflow to run daily and invoke that template.
3. Create a backfill workflow that uses `withSequence` to run the job for each date.

This [full example](https://raw.githubusercontent.com/argoproj/argo-workflows/main/examples/cron-backfill.yaml) contains:

* A workflow template named `job`.
* A cron workflow named `daily-job`.
* A workflow named `backfill-v1` that uses a resource template to create one workflow for each backfill date.
* A alternative workflow named `backfill-v2` that uses a steps templates to run one task for each backfill date.
