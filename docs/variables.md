# Workflow Variables

Some fields in a workflow specification allow for variable references which are automatically substituted by Argo.

## How to use variables

Variables are enclosed in curly braces:

```yaml
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
        command: [ cowsay ]
        args: [ "{{inputs.parameters.message}}" ] 
```

The following variables are made available to reference various metadata of a workflow:

## Template Tag Kinds

There are two kinds of template tag:

* **simple** The default, e.g. `{{workflow.name}}`
* **expression** Where`{{` is immediately followed by `=`, e.g. `{{=workflow.name}}`.

### Simple

The tag is substituted with the variable that has a name the same as the tag.

Simple tags **may** have whitespace between the brackets and variable.

```yaml
args: [ "{{ inputs.parameters.message }}" ]  
```

### Expression

> Since v3.1

The tag is substituted with the result of evaluating the tag as an expression.

Note that any hyphenated parameter names or step names will cause a parsing error. You can reference them by
indexing into the parameter or step map, e.g. `inputs.parameters['my-param']` or `steps['my-step'].outputs.result`.

[Learn about the expression syntax](https://github.com/antonmedv/expr/blob/master/docs/Language-Definition.md).

#### Examples

Plain list:

```
[1, 2]
```

Filter a list:

```
filter([1, 2], { # > 1})
```

Map a list:

```
map([1, 2], { # * 2 })
```

We provide some core functions:

Cast to int:

```
asInt(inputs.parameters['my-int-param'])
```

Cast to float:

```
asFloat(inputs.parameters['my-float-param'])
```

Cast to string:

```
string(1)
```

Convert to a JSON string (needed for `withParam`):

```
toJson([1, 2])
```

Extract data from JSON:

```
jsonpath(inputs.parameters.json, '$.some.path')
```

You can also use [Sprig functions](http://masterminds.github.io/sprig/):

Trim a string:

```
sprig.trim(inputs.parameters['my-string-param'])
```

!!! Warning In Sprig functions, errors are often not raised. E.g. if `int` is used on an invalid value, it
returns `0`. Please review the Sprig documentation to understand which functions do and which do not.

## Reference

### All Templates

| Variable | Description|
|----------|------------|
| `inputs.parameters.<NAME>`| Input parameter to a template |
| `inputs.parameters`| All input parameters to a template as a JSON string |
| `inputs.artifacts.<NAME>` | Input artifact to a template |

### Steps Templates

| Variable | Description|
|----------|------------|
| `steps.<STEPNAME>.id` | unique id of container step |
| `steps.<STEPNAME>.ip` | IP address of a previous daemon container step |
| `steps.<STEPNAME>.status` | Phase status of any previous step |
| `steps.<STEPNAME>.exitCode` | Exit code of any previous script or container step |
| `steps.<STEPNAME>.startedAt` | Timestamp when the step started |
| `steps.<STEPNAME>.finishedAt` | Timestamp when the step finished |
| `steps.<STEPNAME>.outputs.result` | Output result of any previous container or script step |
| `steps.<STEPNAME>.outputs.parameters` | When the previous step uses 'withItems' or 'withParams', this contains a JSON array of the output parameter maps of each invocation |
| `steps.<STEPNAME>.outputs.parameters.<NAME>` | Output parameter of any previous step. When the previous step uses 'withItems' or 'withParams', this contains a JSON array of the output parameter values of each invocation |
| `steps.<STEPNAME>.outputs.artifacts.<NAME>` | Output artifact of any previous step |

### DAG Templates

| Variable | Description|
|----------|------------|
| `tasks.<TASKNAME>.id` | unique id of container task |
| `tasks.<TASKNAME>.ip` | IP address of a previous daemon container task |
| `tasks.<TASKNAME>.status` | Phase status of any previous task |
| `tasks.<TASKNAME>.exitCode` | Exit code of any previous script or container task |
| `tasks.<TASKNAME>.startedAt` | Timestamp when the task started |
| `tasks.<TASKNAME>.finishedAt` | Timestamp when the task finished |
| `tasks.<TASKNAME>.outputs.result` | Output result of any previous container or script task |
| `tasks.<TASKNAME>.outputs.parameters` | When the previous task uses 'withItems' or 'withParams', this contains a JSON array of the output parameter maps of each invocation |
| `tasks.<TASKNAME>.outputs.parameters.<NAME>` | Output parameter of any previous task. When the previous task uses 'withItems' or 'withParams', this contains a JSON array of the output parameter values of each invocation |
| `tasks.<TASKNAME>.outputs.artifacts.<NAME>` | Output artifact of any previous task |

### HTTP Templates

> Since v3.3

Only available for `successCondition`

| Variable | Description|
|----------|------------|
| `request.method` | Request method (`string`) |
| `request.url` | Request URL (`string`) |
| `request.body` | Request body (`string`) |
| `request.headers` | Request headers (`map[string][]string`) |
| `response.statusCode` | Response status code (`int`) |
| `response.body` | Response body (`string`) |
| `response.headers` | Response headers (`map[string][]string`) |

### RetryStrategy

When using the `expression` field within `retryStrategy`, special variables are available.

| Variable | Description|
|----------|------------|
| `lastRetry.exitCode` | Exit code of the last retry |
| `lastRetry.Status` | Status of the last retry |
| `lastRetry.Duration` | Duration in seconds of the last retry |

Note: These variables evaluate to a string type. If using advanced expressions, either cast them to int values (`expression: "{{=asInt(lastRetry.exitCode) >= 2}}"`) or compare them to string values (`expression: "{{=lastRetry.exitCode != '2'}}"`).

### Container/Script Templates

| Variable | Description|
|----------|------------|
| `pod.name` | Pod name of the container/script |
| `retries` | The retry number of the container/script if retryStrategy is specified |
| `inputs.artifacts.<NAME>.path` | Local path of the input artifact |
| `outputs.artifacts.<NAME>.path` | Local path of the output artifact |
| `outputs.parameters.<NAME>.path` | Local path of the output parameter |

### Loops (withItems / withParam)

| Variable | Description|
|----------|------------|
| `item` | Value of the item in a list |
| `item.<FIELDNAME>` | Field value of the item in a list of maps |

### Metrics

When emitting custom metrics in a `template`, special variables are available that allow self-reference to the current
step.

| Variable | Description|
|----------|------------|
| `status` | Phase status of the metric-emitting template |
| `duration` | Duration of the metric-emitting template in seconds (only applicable in `Template`-level metrics, for `Workflow`-level use `workflow.duration`) |
| `exitCode` | Exit code of the metric-emitting template |
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

### Global

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
| `workflow.creationTimestamp.<STRFTIMECHAR>` | Creation timestamp formatted with a [strftime](http://strftime.org) format character. |
| `workflow.creationTimestamp.RFC3339` | Creation timestamp formatted with in RFC 3339. |
| `workflow.priority` | Workflow priority |
| `workflow.duration` | Workflow duration estimate, may differ from actual duration by a couple of seconds |
| `workflow.scheduledTime` | Scheduled runtime formatted in RFC 3339 (only available for CronWorkflows) |

### Exit Handler

| Variable | Description|
|----------|------------|
| `workflow.status` | Workflow status. One of: `Succeeded`, `Failed`, `Error` |
| `workflow.failures` | A list of JSON objects containing information about nodes that failed or errored during execution. Available fields: `displayName`, `message`, `templateName`, `phase`, `podName`, and `finishedAt`. |
