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
  entrypoint: print-message
  arguments:
    parameters:
      - name: message
        value: hello world
  templates:
    - name: print-message
      inputs:
        parameters:
          - name: message
      container:
        image: busybox
        command: [ echo ]
        args: [ "{{inputs.parameters.message}}" ]
```

The following variables are made available to reference various meta-data of a workflow:

## Template Tag Kinds

There are two kinds of template tag:

* **simple** The default, e.g. `{{workflow.name}}`
* **expression** Where`{{` is immediately followed by `=`, e.g. `{{=workflow.name}}`.

### Simple

The tag is substituted with the variable that has a name the same as the tag.

Simple tags **may** have white-space between the brackets and variable as seen below. However, there is a known issue where variables may fail to interpolate with white-space, so it is recommended to avoid using white-space until this issue is resolved. [Please report](https://github.com/argoproj/argo-workflows/issues/8960) unexpected behavior with reproducible examples.

```yaml
args: [ "{{ inputs.parameters.message }}" ]
```

### Expression

> v3.1 and after

The tag is substituted with the result of evaluating the tag as an expression.

Note that any hyphenated parameter names or step names will cause a parsing error. You can reference them by
indexing into the parameter or step map, e.g. `inputs.parameters['my-param']` or `steps['my-step'].outputs.result`.

[Learn more about the expression syntax](https://expr-lang.org/docs/language-definition).

#### Examples

Plain list:

```text
[1, 2]
```

Filter a list:

```text
filter([1, 2], { # > 1})
```

Map a list:

```text
map([1, 2], { # * 2 })
```

Cast to int:

```text
asInt(inputs.parameters['my-int-param'])
```

Cast to float:

```text
asFloat(inputs.parameters['my-float-param'])
```

Cast to string:

```text
string(1)
```

We provide some additional functions:

Convert to a JSON string (needed for `withParam`):

```text
toJson([1, 2])
```

`toJson` is the same as [expr's built-in `toJSON` function](https://expr-lang.org/docs/language-definition#toJSON),
except `toJson` does not add indentation.

Extract data from JSON:

```text
jsonpath(inputs.parameters.json, '$.some.path')
```

#### Sprig Functions

You can also use a curated set of [Sprig functions](http://masterminds.github.io/sprig/):

```text
sprig.trim(inputs.parameters['my-string-param'])
```

!!! Warning "Sprig error handling"
    Sprig functions often do not raise errors.
    For example, if `int` is used on an invalid value, it returns `0`.
    Please review the Sprig documentation to understand which functions raise errors and which do not.

Available Sprig functions include:

* Random/crypto: `randAlpha`, `randAlphaNum`, `randAscii`, `randNumeric`, `randBytes`, `randInt`, `uuidv4`

* Regex helpers: `regexFindAll`, `regexSplit`, `regexReplaceAll`, `regexReplaceAllLiteral`, `regexQuoteMeta`

* Text formatting: `wrap`, `wrapWith`, `nospace`, `title`, `untitle`, `plural`, `initials`, `snakecase`, `camelcase`, `kebabcase`, `swapcase`, `shuffle`, `trunc`

* Dictionary and reflection helpers: `dict`, `set`, `deepCopy`, `merge`, `mergeOverwrite`, `mergeRecursive`, `dig`, `pluck`, `typeIsLike`, `kindIs`, `typeOf`

* Path/URL helpers: `base`, `dir`, `ext`, `clean`, `urlParse`, `urlJoin`

* SemVer helpers: `semver`, `semverCompare`

* Flow control: `fail`, `required`

* Encoding/YAML: `b32enc`, `b32dec`, `toYaml`, `fromYaml`

For complete documentation on these functions, refer to the [Sprig documentation](http://masterminds.github.io/sprig/).

### Migration from Deprecated Sprig Functions

Several Sprig functions that were previously available have been deprecated in favor of Expr standard library alternatives.
While these functions continue to work in v3.7, they will be removed in a future version.
Here's a migration guide for the most commonly used deprecated functions:

| Deprecated Sprig Function | Expr Equivalent | Notes |
|---------------------------|-----------------|-------|
| **String Functions** | | |
| `sprig.toString(value)` | `string(value)` | Direct replacement |
| `sprig.lower(str)` | `lower(str)` | Direct replacement |
| `sprig.upper(str)` | `upper(str)` | Direct replacement |
| `sprig.repeat(str, count)` | `repeat(str, count)` | Direct replacement |
| `sprig.split(delimiter, str)` | `split(str, delimiter)` | Note: parameter order is reversed |
| `sprig.join(delimiter, list)` | `join(list, delimiter)` | Note: parameter order is reversed |
| `sprig.contains(substr, str)` | `indexOf(str, substr) >= 0` | Use indexOf for substring detection |
| `sprig.hasPrefix(prefix, str)` | `hasPrefix(str, prefix)` | Note: parameter order is reversed |
| `sprig.hasSuffix(suffix, str)` | `hasSuffix(str, suffix)` | Note: parameter order is reversed |
| `sprig.replace(old, new, str)` | `replace(str, old, new)` | Note: parameter order is different |
| `sprig.trimSpace(str)` | `trim(str)` | Direct replacement, trims whitespace |
| `sprig.trimLeft(cutset, str)` | No direct equivalent | Use custom logic with substring operations |
| `sprig.trimRight(cutset, str)` | No direct equivalent | Use custom logic with substring operations |
| **Math Functions** | | |
| `sprig.add(a, b)` | `a + b` | Use arithmetic operators |
| `sprig.sub(a, b)` | `a - b` | Use arithmetic operators |
| `sprig.mul(a, b)` | `a * b` | Use arithmetic operators |
| `sprig.div(a, b)` | `a / b` | Use arithmetic operators |
| `sprig.mod(a, b)` | `a % b` | Use arithmetic operators |
| `sprig.max(a, b)` | `max(a, b)` | Direct replacement |
| `sprig.min(a, b)` | `min(a, b)` | Direct replacement |
| `sprig.int(value)` | `int(value)` | Direct replacement |
| `sprig.float64(value)` | `float(value)` | Direct replacement |
| **List Functions** | | |
| `sprig.list(items...)` | `[item1, item2, ...]` | Use array literal syntax |
| `sprig.first(list)` | `list[0]` | Use array indexing |
| `sprig.last(list)` | `list[len(list)-1]` | Use array indexing with length |
| `sprig.rest(list)` | `list[1:]` | Use array slicing |
| `sprig.initial(list)` | `list[:len(list)-1]` | Use array slicing |
| `sprig.reverse(list)` | `reverse(list)` | Direct replacement |
| `sprig.uniq(list)` | No direct equivalent | Use custom filtering logic |
| `sprig.compact(list)` | `filter(list, {# != ""})` | Filter out empty values |
| `sprig.slice(list, start, end)` | `list[start:end]` | Use array slicing |
| **Date/Time Functions** | | |
| `sprig.now()` | `now()` | Direct replacement |
| `sprig.date(layout, time)` | `date(time).Format(layout)` | Use date function with Format method |
| `sprig.dateInZone(layout, time, zone)` | `date(time, zone).Format(layout)` | Use date function with timezone |
| `sprig.unixEpoch(time)` | `date(time).Unix()` | Get Unix timestamp |
| `sprig.dateModify(modifier, time)` | `date(time).Add(duration)` | Use duration arithmetic |
| `sprig.durationRound(duration)` | `duration(duration).Round(precision)` | Use duration methods |
| **Type Conversion** | | |
| `sprig.atoi(str)` | `int(str)` | Direct replacement |
| `sprig.quote(str)` | `"\"" + str + "\""` | Use string concatenation |
| `sprig.squote(str)` | `"'" + str + "'"` | Use string concatenation |
| `sprig.float64(value)` | `float(value)` | Direct replacement |
| `sprig.toString(value)` | `string(value)` | Direct replacement |
| `sprig.toStrings(list)` | `map(list, {string(#)})` | Use map with string conversion |
| **Logic/Flow Control** | | |
| `sprig.and(a, b)` | `a && b` | Use logical operators |
| `sprig.or(a, b)` | `a \|\| b` | Use logical operators |
| `sprig.not(value)` | `!value` | Use logical operator |
| `sprig.eq(a, b)` | `a == b` | Use comparison operators |
| `sprig.ne(a, b)` | `a != b` | Use comparison operators |
| `sprig.lt(a, b)` | `a < b` | Use comparison operators |
| `sprig.le(a, b)` | `a <= b` | Use comparison operators |
| `sprig.gt(a, b)` | `a > b` | Use comparison operators |
| `sprig.ge(a, b)` | `a >= b` | Use comparison operators |
| **Conditionals** | | |
| `sprig.default(default, value)` | `value != "" ? value : default` | Use ternary operator |
| `sprig.empty(value)` | `value == ""` | Use comparison |
| `sprig.ternary(true_val, false_val, condition)` | `condition ? true_val : false_val` | Use ternary operator |
| `sprig.coalesce(vals...)` | `val1 != "" ? val1 : (val2 != "" ? val2 : val3)` | Chain ternary operators |
| **Encoding** | | |
| `sprig.b64enc(str)` | `toBase64(str)` | Direct replacement |
| `sprig.b64dec(str)` | `fromBase64(str)` | Direct replacement |
| **Network** | | |
| `sprig.getHostByName(domain)` | No direct equivalent | Function removed for security |
| **OS/Environment** | | |
| `sprig.env(var)` | No direct equivalent | Function removed for security |
| `sprig.expandenv(str)` | No direct equivalent | Function removed for security |
| **File Path** | | |
| `sprig.base(path)` | No direct equivalent | Use curated sprig function |
| `sprig.dir(path)` | No direct equivalent | Use curated sprig function |
| `sprig.ext(path)` | No direct equivalent | Use curated sprig function |
| `sprig.clean(path)` | No direct equivalent | Use curated sprig function |
| `sprig.isAbs(path)` | No direct equivalent | Limited use in templates |
| **Reflection** | | |
| `sprig.typeOf(value)` | `type(value)` | Direct replacement |
| `sprig.kindOf(value)` | `type(value)` | Similar functionality |
| `sprig.kindIs(kind, value)` | `type(value) == kind` | Use type function with comparison |
| `sprig.typeIs(type, value)` | `type(value) == type` | Use type function with comparison |
| `sprig.typeIsLike(type, value)` | `type(value) == type` | Use type function with comparison |
| `sprig.deepEqual(a, b)` | `a == b` | Use comparison for simple values |
| **JSON Functions** | | |
| `sprig.toJson(value)` | `toJSON(value)` | Direct replacement |
| `sprig.fromJson(str)` | `fromJSON(str)` | Direct replacement |
| **Additional Functions** | | |
| `sprig.get(map, key)` | `get(map, key)` | Direct replacement for safe access |

#### Migration Examples

**String operations:**
```yaml
# Before (deprecated)
args: ["{{=sprig.toString(inputs.parameters.count)}}"]
args: ["{{=sprig.lower(inputs.parameters.name)}}"]
args: ["{{=sprig.replace("foo", "bar", inputs.parameters.text)}}"]

# After (recommended)
args: ["{{=string(inputs.parameters.count)}}"]  
args: ["{{=lower(inputs.parameters.name)}}"]
args: ["{{=replace(inputs.parameters.text, "foo", "bar")}}"]
```

**Math operations:**
```yaml
# Before (deprecated)
args: ["{{=sprig.add(inputs.parameters.a, inputs.parameters.b)}}"]
args: ["{{=sprig.int(inputs.parameters.str_num)}}"]

# After (recommended)  
args: ["{{=int(inputs.parameters.a) + int(inputs.parameters.b)}}"]
args: ["{{=int(inputs.parameters.str_num)}}"]
```

**List operations:**
```yaml
# Before (deprecated)
args: ["{{=sprig.first(myArray)}}"]
args: ["{{=sprig.last(myArray)}}"]
args: ["{{=sprig.join(",", myArray)}}"]
args: ["{{=sprig.reverse(myArray)}}"]
args: ["{{=sprig.compact(myArray)}}"]

# After (recommended)
args: ["{{=myArray[0]}}"]
args: ["{{=myArray[len(myArray)-1]}}"]
# For join, use string concatenation (no direct join function)
args: ["{{=myArray[0] + "," + myArray[1] + "," + myArray[2]}}"]  
# For reverse, access elements in reverse order
args: ["{{=[myArray[2], myArray[1], myArray[0]]}}"]  
# For compact, filter out empty values manually
args: ["{{=myArray[0] != "" ? myArray[0] : (myArray[1] != "" ? myArray[1] : myArray[2])}}"]
```

**Logic and comparison operations:**
```yaml
# Before (deprecated)
condition: "{{=sprig.and(sprig.eq(inputs.parameters.status, "ready"), sprig.gt(inputs.parameters.count, 0))}}"
condition: "{{=sprig.or(sprig.empty(inputs.parameters.value), sprig.eq(inputs.parameters.force, "true"))}}"

# After (recommended)
condition: "{{=inputs.parameters.status == "ready" && int(inputs.parameters.count) > 0}}"
condition: "{{=inputs.parameters.value == "" || inputs.parameters.force == "true"}}"
```

**Conditional logic:**
```yaml
# Before (deprecated)
args: ["{{=sprig.default("unknown", inputs.parameters.name)}}"]
args: ["{{=sprig.ternary("enabled", "disabled", sprig.eq(inputs.parameters.active, "true"))}}"]

# After (recommended)
args: ["{{=inputs.parameters.name != "" ? inputs.parameters.name : "unknown"}}"]
args: ["{{=inputs.parameters.active == "true" ? "enabled" : "disabled"}}"]
```

**Type conversions:**
```yaml
# Before (deprecated)
args: ["{{=sprig.toString(inputs.parameters.count)}}"]
args: ["{{=sprig.float64(inputs.parameters.ratio)}}"]

# After (recommended)
args: ["{{=string(inputs.parameters.count)}}"]
args: ["{{=float(inputs.parameters.ratio)}}"]
```

**Date/time operations:**
```yaml
# Before (deprecated)
args: ["{{=sprig.date("2006-01-02", sprig.now())}}"]
args: ["{{=sprig.dateInZone("15:04:05", sprig.now(), "UTC")}}"]
args: ["{{=sprig.unixEpoch(sprig.now())}}"]
args: ["{{=sprig.dateModify("+24h", sprig.now())}}"]

# After (recommended)
args: ["{{=now().Format("2006-01-02")}}"]
args: ["{{=now().Format("15:04:05")}}"]  # Note: timezone functions not available in expr
args: ["{{=now().Unix()}}"]
args: ["{{=now().Format("2006-01-02")}}"]  # Note: date arithmetic not available, use current time
```

**Common time formatting patterns:**
```yaml
# ISO 8601 date
args: ["{{=now().Format("2006-01-02T15:04:05Z07:00")}}"]

# Human readable date
args: ["{{=now().Format("January 2, 2006")}}"]

# Log timestamp
args: ["{{=now().Format("2006-01-02 15:04:05")}}"]

# File-safe timestamp
args: ["{{=now().Format("20060102-150405")}}"]

# Unix timestamp as string
args: ["{{=string(now().Unix())}}"]

# Workflow creation time access (string format)
args: ["{{=workflow.creationTimestamp}}"]

# Current time basic formats
args: ["{{=now().Format("15:04:05")}}"]
```

!!! Note "Parameter Order Changes"
    Many Expr built-in functions have different parameter orders compared to their Sprig equivalents.
    Always check the parameter order when migrating.
    For example: `sprig.contains(substr, str)` becomes `indexOf(str, substr) >= 0`.

!!! Note "Go Time Formatting"
    Expr uses Go's time formatting, which uses a reference time: `Mon Jan 2 15:04:05 MST 2006` (Unix time `1136239445`).
    This corresponds to `01/02 03:04:05PM '06 -0700`.
    Common format patterns:
    
    - `2006-01-02` = YYYY-MM-DD
    - `15:04:05` = HH:MM:SS (24-hour)
    - `3:04:05 PM` = H:MM:SS AM/PM (12-hour)
    - `January 2, 2006` = Month D, YYYY
    - `02/01/06` = MM/DD/YY
    
    See the [Go time package documentation](https://golang.org/pkg/time/#Time.Format) for more formatting options.

## Reference

### All Templates

| Variable | Description|
|----------|------------|
| `inputs.parameters.<NAME>`| Input parameter to a template |
| `inputs.parameters`| All input parameters to a template as a JSON string |
| `inputs.artifacts.<NAME>` | Input artifact to a template |
| `node.name` | Full name of the node |

### Steps Templates

| Variable | Description|
|----------|------------|
| `steps.name` | Name of the step |
| `steps.<STEPNAME>.id` | unique id of container step |
| `steps.<STEPNAME>.ip` | IP address of a previous daemon container step |
| `steps.<STEPNAME>.status` | Phase status of any previous step |
| `steps.<STEPNAME>.exitCode` | Exit code of any previous script or container step |
| `steps.<STEPNAME>.startedAt` | Time-stamp when the step started |
| `steps.<STEPNAME>.finishedAt` | Time-stamp when the step finished |
| `steps.<TASKNAME>.hostNodeName` | Host node where task ran (available from version 3.5) |
| `steps.<STEPNAME>.outputs.result` | Output result of any previous container, script, or HTTP step |
| `steps.<STEPNAME>.outputs.parameters` | When the previous step uses `withItems` or `withParams`, this contains a JSON array of the output parameter maps of each invocation |
| `steps.<STEPNAME>.outputs.parameters.<NAME>` | Output parameter of any previous step. When the previous step uses `withItems` or `withParams`, this contains a JSON array of the output parameter values of each invocation |
| `steps.<STEPNAME>.outputs.artifacts.<NAME>` | Output artifact of any previous step |

### DAG Templates

| Variable | Description|
|----------|------------|
| `tasks.name` | Name of the task |
| `tasks.<TASKNAME>.id` | unique id of container task |
| `tasks.<TASKNAME>.ip` | IP address of a previous daemon container task |
| `tasks.<TASKNAME>.status` | Phase status of any previous task |
| `tasks.<TASKNAME>.exitCode` | Exit code of any previous script or container task |
| `tasks.<TASKNAME>.startedAt` | Time-stamp when the task started |
| `tasks.<TASKNAME>.finishedAt` | Time-stamp when the task finished |
| `tasks.<TASKNAME>.hostNodeName` | Host node where task ran (available from version 3.5) |
| `tasks.<TASKNAME>.outputs.result` | Output result of any previous container, script, or HTTP task |
| `tasks.<TASKNAME>.outputs.parameters` | When the previous task uses `withItems` or `withParams`, this contains a JSON array of the output parameter maps of each invocation |
| `tasks.<TASKNAME>.outputs.parameters.<NAME>` | Output parameter of any previous task. When the previous task uses `withItems` or `withParams`, this contains a JSON array of the output parameter values of each invocation |
| `tasks.<TASKNAME>.outputs.artifacts.<NAME>` | Output artifact of any previous task |

### HTTP Templates

> v3.3 and after

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

### CronWorkflows

> v3.6 and after

| Variable | Description|
|----------|------------|
| `cronworkflow.name` | Name of the CronWorkflow (`string`) |
| `cronworkflow.namespace` | Namespace of the CronWorkflow (`string`) |
| `cronworkflow.labels.<NAME>` | CronWorkflow labels (`string`) |
| `cronworkflow.labels.json` | CronWorkflow labels as a JSON string (`string`) |
| `cronworkflow.annotations.<NAME>` | CronWorkflow annotations (`string`) |
| `cronworkflow.annotations.json` | CronWorkflow annotations as a JSON string (`string`) |
| `cronworkflow.lastScheduledTime` | The time since this workflow was last scheduled, value is nil on first run (`*time.Time`) |
| `cronworkflow.failed` | Counts how many times child workflows failed |
| `cronworkflow.succeeded` | Counts how many times child workflows succeeded |

### `RetryStrategy`

When using the `expression` field within `retryStrategy`, special variables are available.

| Variable | Description|
|----------|------------|
| `lastRetry.exitCode` | Exit code of the last retry |
| `lastRetry.status` | Status of the last retry |
| `lastRetry.duration` | Duration in seconds of the last retry |
| `lastRetry.message` | Message output from the last retry (available from version 3.5) |

Note: These variables evaluate to a string type. If using advanced expressions, either cast them to int values (`expression: "{{=asInt(lastRetry.exitCode) >= 2}}"`) or compare them to string values (`expression: "{{=lastRetry.exitCode != '2'}}"`).

### Container/Script Templates

| Variable | Description|
|----------|------------|
| `pod.name` | Pod name of the container/script |
| `retries` | The retry number of the container/script if `retryStrategy` is specified |
| `lastRetry` | The last retry is a structure that contains the fields `exitCode`, `status`, `duration` and `message` of the last retry |
| `inputs.artifacts.<NAME>.path` | Local path of the input artifact |
| `outputs.artifacts.<NAME>.path` | Local path of the output artifact |
| `outputs.parameters.<NAME>.path` | Local path of the output parameter |

### Loops (`withItems` / `withParam`)

| Variable | Description|
|----------|------------|
| `item` | Value of the item in a list |
| `item.<FIELDNAME>` | Field value of the item in a list of maps |

### Metrics

When emitting custom metrics in a `template`, special variables are available that allow self-reference to the current
step.

| Variable                         | Description                                                                                                                                                                                  |
|----------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `status`                         | Phase status of the metric-emitting template                                                                                                                                                 |
| `duration`                       | Duration of the metric-emitting template in seconds (only applicable in `Template`-level metrics, for `Workflow`-level use `workflow.duration`)                                              |
| `exitCode`                       | Exit code of the metric-emitting template                                                                                                                                                    |
| `inputs.parameters.<NAME>`       | Input parameter of the metric-emitting template                                                                                                                                              |
| `outputs.parameters.<NAME>`      | Output parameter of the metric-emitting template                                                                                                                                             |
| `outputs.result`                 | Output result of the metric-emitting template                                                                                                                                                |
| `resourcesDuration.{cpu,memory}` | Resources duration **in seconds**. Must be one of `resourcesDuration.cpu` or `resourcesDuration.memory`, if available. For more info, see the [Resource Duration](resource-duration.md) doc. |
| `retries`                        | Retried count by retry strategy                                                                                                                                                              |

### Real-Time Metrics

Some variables can be emitted in real-time (as opposed to just when the step/task completes). To emit these variables in
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
| `workflow.mainEntrypoint` | Workflow's initial entrypoint |
| `workflow.serviceAccountName` | Workflow service account name |
| `workflow.uid` | Workflow UID. Useful for setting ownership reference to a resource, or a unique artifact location |
| `workflow.parameters.<NAME>` | Input parameter to the workflow |
| `workflow.parameters` | All input parameters to the workflow as a JSON string (this is deprecated in favor of `workflow.parameters.json` as this doesn't work with expression tags and that does) |
| `workflow.parameters.json` | All input parameters to the workflow as a JSON string |
| `workflow.outputs.parameters.<NAME>` | Global parameter in the workflow |
| `workflow.outputs.artifacts.<NAME>` | Global artifact in the workflow |
| `workflow.annotations.<NAME>` | Workflow annotations |
| `workflow.annotations.json` | all Workflow annotations as a JSON string |
| `workflow.labels.<NAME>` | Workflow labels |
| `workflow.labels.json` | all Workflow labels as a JSON string |
| `workflow.creationTimestamp` | Workflow creation time-stamp formatted in RFC 3339  (e.g. `2018-08-23T05:42:49Z`) |
| `workflow.creationTimestamp.<STRFTIMECHAR>` | Creation time-stamp formatted with a [`strftime`](http://strftime.org) format character. |
| `workflow.creationTimestamp.RFC3339` | Creation time-stamp formatted with in RFC 3339. |
| `workflow.priority` | Workflow priority |
| `workflow.duration` | Workflow duration estimate in seconds, may differ from actual duration by a couple of seconds |
| `workflow.scheduledTime` | Scheduled runtime formatted in RFC 3339 (only available for `CronWorkflow`) |

### Exit Handler

| Variable | Description|
|----------|------------|
| `workflow.status` | Workflow status. One of: `Succeeded`, `Failed`, `Error` |
| `workflow.failures` | A list of JSON objects containing information about nodes that failed or errored during execution. Available fields: `displayName`, `message`, `templateName`, `phase`, `podName`, and `finishedAt`. |

### Knowing where you are

The idea with creating a `WorkflowTemplate` is that they are reusable bits of code you will use in many actual Workflows. Sometimes it is useful to know which workflow you are part of.

`workflow.mainEntrypoint` is one way you can do this. If each of your actual workflows has a differing entrypoint, you can identify the workflow you're part of. Given this use in a `WorkflowTemplate`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: say-main-entrypoint
spec:
  entrypoint: echo
  templates:
  - name: echo
    container:
      image: alpine
      command: [echo]
      args: ["{{workflow.mainEntrypoint}}"]
```

I can distinguish my caller:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: foo-
spec:
  entrypoint: foo
  templates:
    - name: foo
      steps:
      - - name: step
          templateRef:
            name: say-main-entrypoint
            template: echo
```

results in a log of `foo`

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: bar-
spec:
  entrypoint: bar
  templates:
    - name: bar
      steps:
      - - name: step
          templateRef:
            name: say-main-entrypoint
            template: echo
```

results in a log of `bar`

This shouldn't be that helpful in logging, you should be able to identify workflows through other labels in your cluster's log tool, but can be helpful when generating metrics for the workflow for example.
