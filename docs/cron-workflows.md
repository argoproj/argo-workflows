# Cron Workflows

> v2.5 and after

`CronWorkflows` are workflows that run on a schedule.
They are designed to wrap a `workflowSpec` and to mimic the options of Kubernetes `CronJobs`.
In essence, `CronWorkflow` = `Workflow` + some specific cron options.

## `CronWorkflow` Spec

Below is an example `CronWorkflow`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf
spec:
  schedule: "* * * * *"
  concurrencyPolicy: "Replace"
  startingDeadlineSeconds: 0
  workflowSpec:
    entrypoint: date
    templates:
    - name: date
      container:
        image: alpine:3.6
        command: [sh, -c]
        args: ["date; sleep 90"]
```

### `workflowSpec` and `workflowMetadata`

`CronWorkflow.spec.workflowSpec` is the same type as `Workflow.spec`.
It is a template for `Workflow` objects created from it.

The `Workflow` name is generated based on the `CronWorkflow` name.
In the above example it would be similar to `test-cron-wf-tj6fe`.

You can use `CronWorkflow.spec.workflowMetadata` to add `labels` and `annotations`.

### `CronWorkflow` Options

| Option Name                  | Default Value          | Description |
|:----------------------------:|:----------------------:|-------------|
| `schedule`                   | None | [Cron schedule](#cron-schedule-syntax) to run `Workflows`. Example: `5 4 * * *`. Deprecated, use `schedules`. |
| `schedules`                  | None | v3.6 and after: List of [Cron schedules](#cron-schedule-syntax) to run `Workflows`. Example: `5 4 * * *`, `0 1 * * *`. Either `schedule` or `schedules` must be provided. |
| `timezone`                   | Machine timezone       | [IANA Timezone](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) to run `Workflows`. Example: `America/Los_Angeles` |
| `suspend`                    | `false`                | If `true` Workflow scheduling will not occur. Can be set from the CLI, GitOps, or directly |
| `concurrencyPolicy`          | `Allow`                | What to do if multiple `Workflows` are scheduled at the same time. `Allow`: allow all, `Replace`: remove all old before scheduling new, `Forbid`: do not allow any new while there are old  |
| `startingDeadlineSeconds`    | `0`                    | Seconds after [the last scheduled time](#crash-recovery) during which a missed `Workflow` will still be run. |
| `successfulJobsHistoryLimit` | `3`                    | Number of successful `Workflows` to persist |
| `failedJobsHistoryLimit`     | `1`                    | Number of failed `Workflows` to persist |
| `stopStrategy.expression`    | `nil`                  | v3.6 and after: defines if the CronWorkflow should stop scheduling based on an expression, which if present must evaluate to false for the workflow to be created |
| `when`                       | None | v3.6 and after: An optional [expression](walk-through/conditionals.md) which will be evaluated on each cron schedule hit and the workflow will only run if it evaluates to `true` |

### Cron Schedule Syntax

The cron scheduler uses [standard cron syntax](https://en.wikipedia.org/wiki/Cron).
The implementation is the same as `CronJobs`, using [`robfig/cron`](https://pkg.go.dev/github.com/robfig/cron#hdr-CRON_Expression_Format).

### Crash Recovery

If the Controller crashes, you can ensure that any missed schedules still run.

With `startingDeadlineSeconds` you can specify a maximum grace period past the last scheduled time during which it will still run.
For example, if a `CronWorkflow` that runs every minute is last run at 12:05:00, and the controller crashes between 12:05:55 and 12:06:05, then the expected execution time of 12:06:00 would be missed.
However, if `startingDeadlineSeconds` is set to a value greater than 5 (the time passed between the last scheduled time of 12:06:00 and the current time of 12:06:05), then a single instance of the `CronWorkflow` will be executed exactly at 12:06:05.

Currently only a single instance will be executed as a result of setting `startingDeadlineSeconds`.

This setting can also be configured in tandem with `concurrencyPolicy` to achieve more fine-tuned control.

### Daylight Saving

When using `timezone`, [Daylight Saving Time (DST)](https://en.wikipedia.org/wiki/Daylight_saving_time) is taken into account.
Depending on the local time of the scheduled workflow, it will run once, twice, or not at all when the clock moves forward or back.

For example, with `timezone: America/Los_Angeles`:

- +1 hour (DST start) at 2020-03-08 02:00:00:

    **Note:** The schedules between 02:00 a.m. to 02:59 a.m. were skipped on Mar 8th due to the clock being moved forward:

    | cron       | sequence | workflow execution time       |
    |------------|----------|-------------------------------|
    | 59 1 ** * | 1        | 2020-03-08 01:59:00 -0800 PST |
    |            | 2        | 2020-03-09 01:59:00 -0700 PDT |
    |            | 3        | 2020-03-10 01:59:00 -0700 PDT |
    | 0 2 ** *  | 1        | 2020-03-09 02:00:00 -0700 PDT |
    |            | 2        | 2020-03-10 02:00:00 -0700 PDT |
    |            | 3        | 2020-03-11 02:00:00 -0700 PDT |
    | 1 2 ** *  | 1        | 2020-03-09 02:01:00 -0700 PDT |
    |            | 2        | 2020-03-10 02:01:00 -0700 PDT |
    |            | 3        | 2020-03-11 02:01:00 -0700 PDT |

- -1 hour (DST end) at 2020-11-01 02:00:00:

    **Note:** the schedules between 01:00 a.m. to 01:59 a.m. were triggered twice on Nov 1st due to the clock being set back:

    | cron       | sequence | workflow execution time       |
    |------------|----------|-------------------------------|
    | 59 1 ** * | 1        | 2020-11-01 01:59:00 -0700 PDT |
    |            | 2        | 2020-11-01 01:59:00 -0800 PST |
    |            | 3        | 2020-11-02 01:59:00 -0800 PST |
    | 0 2 ** *  | 1        | 2020-11-01 02:00:00 -0800 PST |
    |            | 2        | 2020-11-02 02:00:00 -0800 PST |
    |            | 3        | 2020-11-03 02:00:00 -0800 PST |
    | 1 2 ** *  | 1        | 2020-11-01 02:01:00 -0800 PST |
    |            | 2        | 2020-11-02 02:01:00 -0800 PST |
    |            | 3        | 2020-11-03 02:01:00 -0800 PST |

#### Skip forward (missing schedule)

You can use `when` to schedule once per day, even if the time you want is in a daylight saving skip forward period where it would otherwise be scheduled twice.

An example 02:30:00 schedule

```yaml
schedules:
  - 30 2 * * *
  - 0 3 * * *
when: "{{= cronworkflow.lastScheduledTime == nil || (now() - cronworkflow.lastScheduledTime).Seconds() > 3600 }}"
```

The 3:00 run of the schedule will not be scheduled every day of the year except on the day when the clock leaps forward over 2:30.
The `when` expression prevents this workflow from running more than once an hour.
In that case the 3:00 run will run, as 2:30 was skipped over.

!!! Warning "Can run at 3:00"
 If you create this CronWorkflow between the desired time and 3:00 it will run at 3:00 as it has never run before.

#### Skip backwards (duplication)

You can use `when` to schedule once per day, even if the time you want is in a daylight saving skip backwards period where it would otherwise not be scheduled.

An example 01:30:00 schedule

```yaml
schedules:
  - 30 1 * * *
when: "{{= cronworkflow.lastScheduledTime == nil || (now() - cronworkflow.lastScheduledTime).Seconds() > 7200 }}"
```

This will schedule at the first 01:30 on a skip backwards change.
The second will not run because of the `when` expression, which prevents this workflow running more often than once every 2 hours..

### Automatically Stopping a `CronWorkflow`

> v3.6 and after

You can configure a `CronWorkflow` to automatically stop based on an [expression](variables.md#expression) with `stopStrategy.expression`.
You can use the [variables](variables.md#cronworkflows) `cronworkflow.failed` and `cronworkflow.succeeded`.

For example, if you want to stop scheduling new workflows after one success:

```yaml
stopStrategy:
  expression: "cronworkflow.succeeded >= 1"
```

You can also stop scheduling new workflows after three failures with:

```yaml
stopStrategy:
  expression: "cronworkflow.failed >= 3"
```

<!-- markdownlint-disable MD046 -- this is indented due to the admonition, not a code block -->
!!! Warning "Scheduling vs. Completions"
    Depending on the time it takes to schedule and run a workflow, the number of completions can exceed the configured maximum.

    For example, if you configure the `CronWorkflow` to schedule every minute (`* * * * *`) and stop after one success (`cronworkflow.succeeded >= 1`).
    If the `Workflow` takes 90 seconds to run, the `CronWorkflow` will actually stop after two completions.
    This is because when the stopping condition is achieved, there is _already_ another `Workflow` running.
    For that reason, prefer conditions like `cronworkflow.succeeded >= 1` over `cronworkflow.succeeded == 1`.
<!-- markdownlint-enable MD046 -->

## Managing `CronWorkflow`

### CLI

You can create `CronWorkflows` with the CLI:

```bash
$ argo cron create cron.yaml
Name:                          test-cron-wf
Namespace:                     argo
Created:                       Mon Nov 18 10:17:06 -0800 (now)
Schedules:                     * * * * *
Suspended:                     false
StartingDeadlineSeconds:       0
ConcurrencyPolicy:             Forbid

$ argo cron list
NAME           AGE   LAST RUN   SCHEDULES    SUSPENDED
test-cron-wf   49s   N/A        * * * * *   false

# some time passes

$ argo cron list
NAME           AGE   LAST RUN   SCHEDULES    SUSPENDED
test-cron-wf   56s   2s         * * * * *   false

$ argo cron get test-cron-wf
Name:                          test-cron-wf
Namespace:                     argo
Created:                       Wed Oct 28 07:19:02 -0600 (23 hours ago)
Schedules:                      * * * * *
Suspended:                     false
StartingDeadlineSeconds:       0
ConcurrencyPolicy:             Replace
LastScheduledTime:             Thu Oct 29 06:51:00 -0600 (11 minutes ago)
NextScheduledTime:             Thu Oct 29 13:03:00 +0000 (32 seconds from now)
Active Workflows:              test-cron-wf-rt4nf
```

**Note**: `NextScheduledRun` assumes the Controller uses UTC as its timezone

### `kubectl`

You can use `kubectl apply -f` and `kubectl get cwf`

## Back-Filling Days

See [cron backfill](cron-backfill.md).

### GitOps via Argo CD

You can manage `CronWorkflow` resources with GitOps by using [Argo CD](https://github.com/argoproj/argo-cd)

### UI

You can also manage `CronWorkflow` resources in the UI
