# Cron Workflows

![GA](assets/ga.svg)

> v2.5 and after

## Introduction

`CronWorkflow` are workflows that run on a preset schedule. They are designed to be converted from `Workflow` easily and to mimick the same options as Kubernetes `CronJob`. In essence, `CronWorkflow` = `Workflow` + some specific cron options.

## `CronWorkflow` Spec

An example `CronWorkflow` spec would look like:

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
    entrypoint: whalesay
    templates:
    - name: whalesay
      container:
        image: alpine:3.6
        command: [sh, -c]
        args: ["date; sleep 90"]
```

### `workflowSpec` and `workflowMetadata`

`CronWorkflow.spec.workflowSpec` is the same type as `Workflow.spec` and serves as a template for `Workflow` objects that are created from it. Everything under this spec will be converted to a `Workflow`.

The resuling `Workflow` name will be a generated name based on the `CronWorkflow` name. In this example it could be something like `test-cron-wf-tj6fe`.

`CronWorkflow.spec.workflowMetadata` can be used to add `labels` and `annotations`.

### `CronWorkflow` Options

|          Option Name         |      Default Value     | Description                                                                                                                                                                                                                             |
|:----------------------------:|:----------------------:|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
|          `schedule`          | None, must be provided | Schedule at which the `Workflow` will be run. E.g. `5 4 * * * `                                                                                                                                                                         |
|          `timezone`          |    Machine timezone    | Timezone during which the Workflow will be run from the IANA timezone standard, e.g. `America/Los_Angeles`                                                                                                                              |
|           `suspend`          |         `false`        | If `true` Workflow scheduling will not occur. Can be set from the CLI, GitOps, or directly                                                                                                                                              |
|      `concurrencyPolicy`     |         `Allow`        | Policy that determines what to do if multiple `Workflows` are scheduled at the same time. Available options: `Allow`: allow all, `Replace`: remove all old before scheduling a new, `Forbid`: do not allow any new while there are old  |
| `startingDeadlineSeconds`    |           `0`          | Number of seconds after the last successful run during which a missed `Workflow` will be run                                                                                                                                            |
| `successfulJobsHistoryLimit` |           `3`          | Number of successful `Workflows` that will be persisted at a time                                                                                                                                                                       |
| `failedJobsHistoryLimit`     | `1`                    | Number of failed `Workflows` that will be persisted at a time                                                                                                                                                                           |

### Crash Recovery

If the `workflow-controller` crashes (and hence the `CronWorkflow` controller), there are some options you can set to ensure that `CronWorkflows` that would have been scheduled while the controller was down can still run. Mainly `startingDeadlineSeconds` can be set to specify the maximum number of seconds past the last successful run of a `CronWorkflow` during which a missed run will still be executed.

For example, if a `CronWorkflow` that runs every minute is last run at 12:05:00, and the controller crashes between 12:05:55 and 12:06:05, then the expected execution time of 12:06:00 would be missed. However, if `startingDeadlineSeconds` is set to a value greater than 65 (the amount of time passing between the last scheduled run time of 12:05:00 and the current controller restart time of 12:06:05), then a single instance of the `CronWorkflow` will be executed exactly at 12:06:05.

Currently only a single instance will be executed as a result of setting `startingDeadlineSeconds`.

This setting can also be configured in tandem with `concurrencyPolicy` to achieve more fine-tuned control.

## Managing `CronWorkflow`

### CLI

`CronWorkflow` can be created from the CLI by using basic commands:

```sh
$ argo cron create cron.yaml
Name:                          test-cron-wf
Namespace:                     argo
Created:                       Mon Nov 18 10:17:06 -0800 (now)
Schedule:                      * * * * *
Suspended:                     false
StartingDeadlineSeconds:       0
ConcurrencyPolicy:             Forbid

$ argo cron list
NAME           AGE   LAST RUN   SCHEDULE    SUSPENDED
test-cron-wf   49s   N/A        * * * * *   false

# some time passes

$ argo cron list
NAME           AGE   LAST RUN   SCHEDULE    SUSPENDED
test-cron-wf   56s   2s         * * * * *   false

$ argo cron get test-cron-wf
Name:                          test-cron-wf
Namespace:                     argo
Created:                       Wed Oct 28 07:19:02 -0600 (23 hours ago)
Schedule:                      * * * * *
Suspended:                     false
StartingDeadlineSeconds:       0
ConcurrencyPolicy:             Replace
LastScheduledTime:             Thu Oct 29 06:51:00 -0600 (11 minutes ago)
NextScheduledTime:             Thu Oct 29 13:03:00 +0000 (32 seconds from now)
Active Workflows:              test-cron-wf-rt4nf
```

**Note**: `NextScheduledRun` assumes that the workflow-controller uses UTC as its timezone

### `kubectl`

Using `kubectl apply -f` and `kubectl get cwf`

## Backfilling Days

See [cron backfill](cron-backfill.md).

### GitOps via Argo CD

`CronWorkflow` resources can be managed with GitOps by using [Argo CD](https://github.com/argoproj/argo-cd)

### UI

`CronWorkflow` resources can also be managed by the UI
