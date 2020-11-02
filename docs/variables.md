# Workflow Variables

Some fields in a workflow specification allow for variable references which are automatically substituted by Argo.

??? note "How to use variables"
    Variables are enclosed in curly braces and **may** include whitespace between the brackets and variable.

    ``` yaml
    apiVersion: argoproj.io/v1alpha1
    kind: Workflow
    metadata:
      generateName: hello-world-parameters-
    spec:
      entrypoint: whalesay
      arguments:
        parameters:
        - name: message
          value: hello world
      templates:
      - name: whalesay
        inputs:
          parameters:
          - name: message
        container:
          image: docker/whalesay
          command: [cowsay]
        # args: ["{{ inputs.parameters.message }}"]       <- good
          args: ["{{inputs.parameters.message}}"]         #  good
    ```

The following variables are made available to reference various metadata of a workflow:

## All Templates
| Variable | Description|
|----------|------------|
| `inputs.parameters.<NAME>`| Input parameter to a template |
| `inputs.parameters`| All input parameters to a template as a JSON string |
| `inputs.artifacts.<NAME>` | Input artifact to a template |

## Steps Templates
| Variable | Description|
|----------|------------|
| `steps.<STEPNAME>.id` | unique id of container step |
| `steps.<STEPNAME>.ip` | IP address of a previous daemon container step |
| `steps.<STEPNAME>.status` | Phase status of any previous step |
| `steps.<STEPNAME>.exitCode` | Exit code of any previous script or container step |
| `steps.<STEPNAME>.startedAt` | Timestamp when the step started |
| `steps.<STEPNAME>.finishedAt` | Timestamp when the step finished |
| `steps.<STEPNAME>.outputs.result` | Output result of any previous container or script step |
| `steps.<STEPNAME>.outputs.parameters` | When the previous step uses 'withItems', this contains a JSON array of the output parameters of each invocation |
| `steps.<STEPNAME>.outputs.parameters.<NAME>` | Output parameter of any previous step |
| `steps.<STEPNAME>.outputs.artifacts.<NAME>` | Output artifact of any previous step |

## DAG Templates
| Variable | Description|
|----------|------------|
| `tasks.<TASKNAME>.id` | unique id of container task |
| `tasks.<TASKNAME>.ip` | IP address of a previous daemon container task |
| `tasks.<TASKNAME>.status` | Phase status of any previous task |
| `tasks.<TASKNAME>.exitCode` | Exit code of any previous script or container task |
| `tasks.<TASKNAME>.startedAt` | Timestamp when the task started |
| `tasks.<TASKNAME>.finishedAt` | Timestamp when the task finished |
| `tasks.<TASKNAME>.outputs.result` | Output result of any previous container or script task |
| `tasks.<TASKNAME>.outputs.parameters` | When the previous task uses 'withItems', this contains a JSON array of the output parameters of each invocation |
| `tasks.<TASKNAME>.outputs.parameters.<NAME>` | Output parameter of any previous task |
| `tasks.<TASKNAME>.outputs.artifacts.<NAME>` | Output artifact of any previous task |

## Container/Script Templates
| Variable | Description|
|----------|------------|
| `pod.name` | Pod name of the container/script |
| `retries` | The retry number of the container/script if retryStrategy is specified |
| `inputs.artifacts.<NAME>.path` | Local path of the input artifact |
| `outputs.artifacts.<NAME>.path` | Local path of the output artifact |
| `outputs.parameters.<NAME>.path` | Local path of the output parameter |

## Loops (withItems / withParam)
| Variable | Description|
|----------|------------|
| `item` | Value of the item in a list |
| `item.<FIELDNAME>` | Field value of the item in a list of maps |

## Metrics
When emitting custom metrics in a `template`, special variables are available that allow self-reference to the current
step.

| Variable | Description|
|----------|------------|
| `status` | Phase status of the metric-emitting template |
| `duration` | Duration of the metric-emitting template in seconds (only applicable in `Template`-level metrics, for `Workflow`-level use `workflow.duration`) |
| `inputs.parameters.<NAME>` | Input parameter of the metric-emitting template |
| `outputs.parameters.<NAME>` | Output parameter of the metric-emitting template |
| `outputs.result` | Output result of the metric-emitting template |
| `resourcesDuration.{cpu,memory}` | Resources duration **in seconds**. Must be one of `resourcesDuration.cpu` or `resourcesDuration.memory`, if available. For more info, see the [Resource Duration](resource-duration.md) doc.|

### Realtime Metrics

Some variables can be emitted in realtime (as opposed to just when the step/task completes). To emit these variables in
real time, set `realtime: true` under `gauge` (note: only Gauge metrics allow for real time variable emission). Metrics
currently available for real time emission:

For `Workflow`-level metrics:
* `workflow.duration`

For `Template`-level metrics:
* `duration`

## Global
| Variable | Description|
|----------|------------|
| `workflow.name` | Workflow name |
| `workflow.namespace` | Workflow namespace |
| `workflow.serviceAccountName` | Workflow service account name |
| `workflow.uid` | Workflow UID. Useful for setting ownership reference to a resource, or a unique artifact location |
| `workflow.parameters.<NAME>` | Input parameter to the workflow |
| `workflow.parameters` | All input parameters to the workflow as a JSON string |
| `workflow.outputs.parameters.<NAME>` | Global parameter in the workflow |
| `workflow.outputs.artifacts.<NAME>` | Global artifact in the workflow |
| `workflow.annotations.<NAME>` | Workflow annotations |
| `workflow.labels.<NAME>` | Workflow labels |
| `workflow.creationTimestamp` | Workflow creation timestamp formatted in RFC 3339  (e.g. `2018-08-23T05:42:49Z`) |
| `workflow.creationTimestamp.<STRFTIMECHAR>` | Creation timestamp formatted with a [strftime](http://strftime.org) format character |
| `workflow.priority` | Workflow priority |
| `workflow.duration` | Workflow duration estimate, may differ from actual duration by a couple of seconds |

## Exit Handler
| Variable | Description|
|----------|------------|
| `workflow.status` | Workflow status. One of: `Succeeded`, `Failed`, `Error` |
| `workflow.failures` | A list of JSON objects containing information about nodes that failed or errored during execution. Available fields: `displayName`, `message`, `templateName`, `phase`, `podName`, and `finishedAt`. |
