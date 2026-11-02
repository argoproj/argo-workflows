# Cron Backfill

You can backfill in two ways:

* Run [`argo cron backfill`](#backfill-with-the-cli), available in v3.7 and later.
* Write [a workflow that runs the backfill](#backfill-with-a-workflow).

## Use Case

* You are using cron workflows to run daily jobs, you may need to re-run for a date, or run some historical days.

## Backfill with the CLI

`argo cron backfill` creates one workflow for each time the CronWorkflow's schedule would have run between a start and an end time.
Each workflow receives its scheduled time as a parameter.
The [CLI reference](cli/argo_cron_backfill.md) lists every flag and its default.

### Prepare Your CronWorkflow

Put the templates in a WorkflowTemplate and reference it from the CronWorkflow with `workflowTemplateRef`.
The command copies `spec.workflowSpec` into a resource template manifest in another workflow, the backfill workflow.
The backfill workflow fails validation if the manifest references input, step, task or item variables, or workflow parameters, such as `{{inputs.parameters.date}}`.
Global variables such as `{{workflow.name}}` pass validation, but resolve to the backfill workflow's values.

Declare the parameter that receives the scheduled time, named by `--argname`, in the WorkflowTemplate's `spec.arguments`, with a default value for scheduled runs.
Don't declare it in the CronWorkflow's `spec.workflowSpec.arguments`: the command adds the parameter there, and a duplicate name fails validation.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: daily-job
spec:
  entrypoint: main
  arguments:
    parameters:
      - name: date
        value: today
  templates:
    - name: main
      inputs:
        parameters:
          - name: date
      script:
        image: busybox
        command: [sh]
        source: |
          date="{{inputs.parameters.date}}"
          if [ "$date" = today ]; then
            date=$(date -u)
          fi
          echo "run for $date"
---
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: daily-job
spec:
  schedules:
    - "0 2 * * *"
  workflowSpec:
    workflowTemplateRef:
      name: daily-job
```

### Run the Backfill

```bash
argo cron backfill daily-job --name oct2024 --argname date \
  --start "Mon, 21 Oct 2024 00:00:00 UTC" \
  --end "Thu, 24 Oct 2024 00:00:00 UTC"
```

The schedule runs at 02:00, so this creates three workflows, for 02:00 on 21, 22 and 23 October.
Each one receives a `date` parameter such as `2024-10-21 02:00:00 +0000 UTC`.
The backfill period excludes a run scheduled exactly at `--start` and includes one scheduled exactly at `--end`.

```text
Created oct2024 Backfill task for Cronworkflow daily-job
==================================================
Backfill Period :
Start Time : Mon, 21 Oct 2024 00:00:00 UTC
  End Time : Thu, 24 Oct 2024 00:00:00 UTC
Total Backfill Schedule: 3
==================================================
Backfill Workflows:
   NAMESPACE     WORKFLOW:
1. argo          backfill-wf-7xk2p
```

### Scheduled Times and Time Zones

The command ignores `spec.timezone` and evaluates the schedule in the time zone of `--start`.
Give `--start` and `--end` in the time zone you want the schedule to run in.

With the default `--format`, use `UTC` or `GMT` as the zone.
Go reads other abbreviations, such as `EST`, as an offset of `+0000` unless the abbreviation matches the time zone of the machine you run the command on.
To give an offset instead, change the layout, for example to RFC 3339:

```bash
argo cron backfill daily-job --argname date \
  --format "2006-01-02T15:04:05Z07:00" \
  --start 2024-10-21T00:00:00-05:00 \
  --end 2024-10-24T00:00:00-05:00
```

The parameter value always uses Go's default time format, `2006-01-02 15:04:05 -0700 MST`, whatever you set `--format` to.
In a backfilled workflow `{{workflow.scheduledTime}}` is the time you ran the command, so read the scheduled time from the parameter.

The command supports only a CronWorkflow with a single entry in `spec.schedules`, and fails to parse the schedule if there are more.
It doesn't evaluate `spec.when`, and it runs whether or not the CronWorkflow is suspended.

### How It Works

The command creates a backfill workflow named `backfill-wf-<random>` with the label `workflows.argoproj.io/backfill: <cron-workflow-name>`.
For each scheduled time, a step in the backfill workflow creates a workflow named `<cron-workflow-name>-backfill-<name>-<random>` and waits for it to succeed.
`<name>` is the value of `--name`, or a random five-character string if you don't set it.
The steps start together, each in its own pod, and each pod stays up until its workflow succeeds.

If there are more scheduled times than `--maxworkflowcount`, the command creates several backfill workflows, each with at most `--maxworkflowcount` steps.
The flag doesn't cap the total number of workflows the command creates.

Without `--parallel`, every backfilled workflow holds a [mutex](synchronization.md) named after `--name`, so they run one at a time.
They don't necessarily run in date order.
With `--parallel`, they all run at once, subject to your [parallelism](parallelism.md) settings.

To see a backfill's progress:

```bash
argo list -l workflows.argoproj.io/backfill=daily-job
```

The backfill workflow runs with the namespace's default service account, or the one set in your [default workflow spec](default-workflow-specs.md).
That service account needs permission to create and get workflows, as well as the [permissions every workflow needs](workflow-rbac.md).

If a backfilled workflow fails, its step keeps waiting, and the backfill workflow never finishes.
Stop it with `argo stop`; the workflows it has already created keep running.

### Interaction with the CronWorkflow

Backfilled workflows carry the `workflows.argoproj.io/cron-workflow` label and an owner reference to the CronWorkflow, so the CronWorkflow treats them as its own runs:

* They count towards `successfulJobsHistoryLimit` and `failedJobsHistoryLimit`, which default to 3 and 1.
  The controller deletes the oldest finished workflows beyond these limits, backfilled ones included.
* With `concurrencyPolicy: Forbid`, the controller skips a scheduled run while any backfilled workflow is running.
* With `concurrencyPolicy: Replace`, the next scheduled run terminates any running backfilled workflows.
* Deleting the CronWorkflow deletes the backfilled workflows.

## Backfill with a Workflow

1. Create a workflow template for your daily job.
2. Create your cron workflow to run daily and invoke that template.
3. Create a backfill workflow that uses `withSequence` to run the job for each date.

This [full example](https://raw.githubusercontent.com/argoproj/argo-workflows/main/examples/cron-backfill.yaml) contains:

* A workflow template named `job`.
* A cron workflow named `daily-job`.
* A workflow named `backfill-v1` that uses a resource template to create one workflow for each backfill date.
* An alternative workflow named `backfill-v2` that uses a steps template to run one step for each backfill date.
