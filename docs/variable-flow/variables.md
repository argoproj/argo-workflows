# Workflow variables catalog

Auto-generated from `util/variables` via `GenerateMarkdown()`. 68 variables registered.

## 1. Alphabetical index

|                    Key                    |   Kind   |     Type      |                        Availability                        |                                       Description                                       |
|-------------------------------------------|----------|---------------|------------------------------------------------------------|-----------------------------------------------------------------------------------------|
| `inputs.artifacts.<name>`                 | input    | wfv1.Artifact | during-execute                                             | Input artifact object (for fromExpression use)                                          |
| `inputs.artifacts.<name>.path`            | input    | string        | during-execute                                             | Mount path of the input artifact inside the pod                                         |
| `inputs.parameters`                       | input    | json          | during-execute                                             | All input parameters as a JSON array                                                    |
| `inputs.parameters.<name>`                | input    | string        | during-execute                                             | Resolved input parameter value                                                          |
| `item`                                    | item     | string|json   | inside-loop, during-execute                                | Current loop iteration value (withItems/withParam). JSON for map/list items.            |
| `item.<key>`                              | item     | string        | inside-loop, during-execute                                | Accessor into a map-typed loop iteration value                                          |
| `node.name`                               | node-ctx | string        | pre-dispatch, during-execute                               | Full node name                                                                          |
| `outputs.artifacts.<name>.path`           | input    | string        | during-execute                                             | Declared output artifact path for the current template (pod side)                       |
| `outputs.parameters.<name>.path`          | input    | string        | during-execute                                             | Declared output parameter path for the current template (pod side)                      |
| `pod.name`                                | node-ctx | string        | pre-dispatch, during-execute                               | Computed pod name for pod-producing templates                                           |
| `retries`                                 | retry    | string        | inside-retry, during-execute                               | 0-based retry attempt index                                                             |
| `retries.lastDuration`                    | retry    | string        | inside-retry, during-execute                               | Duration of the previous attempt in seconds                                             |
| `retries.lastExitCode`                    | retry    | string        | inside-retry, during-execute                               | Exit code of the previous attempt (or 0 on first attempt)                               |
| `retries.lastMessage`                     | retry    | string        | inside-retry, during-execute                               | Message of the previous attempt                                                         |
| `retries.lastStatus`                      | retry    | string        | inside-retry, during-execute                               | Phase of the previous attempt (or empty on first)                                       |
| `steps.<loopName>.outputs.parameters`     | node-ref | json          | after-loop                                                 | JSON array of per-child output-parameter maps                                           |
| `steps.<loopName>.outputs.parameters.<p>` | node-ref | json          | after-loop                                                 | JSON array of values for a named parameter across all children                          |
| `steps.<loopName>.outputs.result`         | node-ref | json          | after-loop                                                 | JSON array of child results (withItems/withParam)                                       |
| `steps.<name>.exitCode`                   | node-ref | string        | after-node-complete                                        | Container exit code                                                                     |
| `steps.<name>.finishedAt`                 | node-ref | string        | after-node-complete                                        | RFC3339 finish time                                                                     |
| `steps.<name>.hostNodeName`               | node-ref | string        | after-pod-start                                            | Underlying k8s node name                                                                |
| `steps.<name>.id`                         | node-ref | string        | after-node-init                                            | Node ID                                                                                 |
| `steps.<name>.ip`                         | node-ref | string        | after-pod-start                                            | Pod IP                                                                                  |
| `steps.<name>.outputs.artifacts.<a>`      | node-ref | wfv1.Artifact | after-node-succeeded                                       | Named output artifact of the referenced node                                            |
| `steps.<name>.outputs.parameters.<p>`     | node-ref | string        | after-node-succeeded                                       | Named output parameter of the referenced node                                           |
| `steps.<name>.outputs.result`             | node-ref | string        | after-node-succeeded                                       | Captured stdout (non-loop nodes)                                                        |
| `steps.<name>.startedAt`                  | node-ref | string        | after-pod-start                                            | RFC3339 start time                                                                      |
| `steps.<name>.status`                     | node-ref | string        | after-node-init                                            | Node phase                                                                              |
| `steps.name`                              | node-ctx | string        | pre-dispatch, during-execute                               | Name of the current step (inside a Steps template body)                                 |
| `tasks.<loopName>.outputs.parameters`     | node-ref | json          | after-loop                                                 | JSON array of per-child output-parameter maps                                           |
| `tasks.<loopName>.outputs.parameters.<p>` | node-ref | json          | after-loop                                                 | JSON array of values for a named parameter across all children                          |
| `tasks.<loopName>.outputs.result`         | node-ref | json          | after-loop                                                 | JSON array of child results (withItems/withParam)                                       |
| `tasks.<name>.exitCode`                   | node-ref | string        | after-node-complete                                        | Container exit code                                                                     |
| `tasks.<name>.finishedAt`                 | node-ref | string        | after-node-complete                                        | RFC3339 finish time                                                                     |
| `tasks.<name>.hostNodeName`               | node-ref | string        | after-pod-start                                            | Underlying k8s node name                                                                |
| `tasks.<name>.id`                         | node-ref | string        | after-node-init                                            | Node ID                                                                                 |
| `tasks.<name>.ip`                         | node-ref | string        | after-pod-start                                            | Pod IP                                                                                  |
| `tasks.<name>.outputs.artifacts.<a>`      | node-ref | wfv1.Artifact | after-node-succeeded                                       | Named output artifact of the referenced node                                            |
| `tasks.<name>.outputs.parameters.<p>`     | node-ref | string        | after-node-succeeded                                       | Named output parameter of the referenced node                                           |
| `tasks.<name>.outputs.result`             | node-ref | string        | after-node-succeeded                                       | Captured stdout (non-loop nodes)                                                        |
| `tasks.<name>.startedAt`                  | node-ref | string        | after-pod-start                                            | RFC3339 start time                                                                      |
| `tasks.<name>.status`                     | node-ref | string        | after-node-init                                            | Node phase                                                                              |
| `tasks.name`                              | node-ctx | string        | pre-dispatch, during-execute                               | Name of the current task (inside a DAG template body)                                   |
| `workflow.annotations`                    | global   | json          | workflow-start, pre-dispatch, during-execute, exit-handler | All workflow annotations as a JSON object (deprecated — use workflow.annotations.json)  |
| `workflow.annotations.<name>`             | global   | string        | workflow-start, pre-dispatch, during-execute, exit-handler | Workflow metadata annotation value                                                      |
| `workflow.annotations.json`               | global   | json          | workflow-start, pre-dispatch, during-execute, exit-handler | All workflow annotations as a JSON object                                               |
| `workflow.creationTimestamp`              | global   | string        | workflow-start, pre-dispatch, during-execute, exit-handler | RFC3339 creation timestamp                                                              |
| `workflow.creationTimestamp.<fmt>`        | global   | string        | workflow-start, pre-dispatch, during-execute, exit-handler | strftime-formatted workflow creation time; `<fmt>` is one of the chars in util/strftime |
| `workflow.creationTimestamp.RFC3339`      | global   | string        | workflow-start, pre-dispatch, during-execute, exit-handler | Workflow creation time as RFC3339                                                       |
| `workflow.creationTimestamp.s`            | global   | string        | workflow-start, pre-dispatch, during-execute, exit-handler | Workflow creation time as Unix seconds                                                  |
| `workflow.duration`                       | runtime  | string        | pre-dispatch, during-execute, exit-handler                 | Elapsed seconds as float string; final at exit handler                                  |
| `workflow.failures`                       | runtime  | json          | exit-handler                                               | JSON array of failed node descriptors; populated when any node failed                   |
| `workflow.labels`                         | global   | json          | workflow-start, pre-dispatch, during-execute, exit-handler | All workflow labels as a JSON object (deprecated — use workflow.labels.json)            |
| `workflow.labels.<name>`                  | global   | string        | workflow-start, pre-dispatch, during-execute, exit-handler | Workflow metadata label value                                                           |
| `workflow.labels.json`                    | global   | json          | workflow-start, pre-dispatch, during-execute, exit-handler | All workflow labels as a JSON object                                                    |
| `workflow.mainEntrypoint`                 | global   | string        | workflow-start, pre-dispatch, during-execute, exit-handler | spec.entrypoint                                                                         |
| `workflow.name`                           | global   | string        | workflow-start, pre-dispatch, during-execute, exit-handler | Workflow object name                                                                    |
| `workflow.namespace`                      | global   | string        | workflow-start, pre-dispatch, during-execute, exit-handler | Workflow namespace                                                                      |
| `workflow.outputs.artifacts.<name>`       | node-ref | wfv1.Artifact | during-execute, exit-handler                               | Global output artifact (lifted via outputs.artifacts[*].globalName)                     |
| `workflow.outputs.parameters.<name>`      | node-ref | string        | during-execute, exit-handler                               | Global output parameter (lifted via outputs.parameters[*].globalName)                   |
| `workflow.parameters`                     | global   | json          | workflow-start, pre-dispatch, during-execute, exit-handler | All workflow parameters as a JSON array                                                 |
| `workflow.parameters.<name>`              | global   | string        | workflow-start, pre-dispatch, during-execute, exit-handler | Value from spec.arguments.parameters, ConfigMap-resolved if ValueFrom is set            |
| `workflow.parameters.json`                | global   | json          | workflow-start, pre-dispatch, during-execute, exit-handler | All workflow parameters as a JSON array (alias for workflow.parameters)                 |
| `workflow.priority`                       | global   | string        | workflow-start, pre-dispatch, during-execute, exit-handler | Workflow priority                                                                       |
| `workflow.scheduledTime`                  | global   | string        | workflow-start, pre-dispatch, during-execute, exit-handler | Scheduled time for cron-triggered workflows (from annotation)                           |
| `workflow.serviceAccountName`             | global   | string        | workflow-start, pre-dispatch, during-execute, exit-handler | Effective service account name                                                          |
| `workflow.status`                         | runtime  | string        | pre-dispatch, during-execute, exit-handler                 | Current workflow phase; final value only at exit handler                                |
| `workflow.uid`                            | global   | string        | workflow-start, pre-dispatch, during-execute, exit-handler | Workflow UID                                                                            |

## 2. Grouped by Kind

### Global

|                 Key                  |  Type  |                        Availability                        |                                       Description                                       |
|--------------------------------------|--------|------------------------------------------------------------|-----------------------------------------------------------------------------------------|
| `workflow.annotations`               | json   | workflow-start, pre-dispatch, during-execute, exit-handler | All workflow annotations as a JSON object (deprecated — use workflow.annotations.json)  |
| `workflow.annotations.<name>`        | string | workflow-start, pre-dispatch, during-execute, exit-handler | Workflow metadata annotation value                                                      |
| `workflow.annotations.json`          | json   | workflow-start, pre-dispatch, during-execute, exit-handler | All workflow annotations as a JSON object                                               |
| `workflow.creationTimestamp`         | string | workflow-start, pre-dispatch, during-execute, exit-handler | RFC3339 creation timestamp                                                              |
| `workflow.creationTimestamp.<fmt>`   | string | workflow-start, pre-dispatch, during-execute, exit-handler | strftime-formatted workflow creation time; `<fmt>` is one of the chars in util/strftime |
| `workflow.creationTimestamp.RFC3339` | string | workflow-start, pre-dispatch, during-execute, exit-handler | Workflow creation time as RFC3339                                                       |
| `workflow.creationTimestamp.s`       | string | workflow-start, pre-dispatch, during-execute, exit-handler | Workflow creation time as Unix seconds                                                  |
| `workflow.labels`                    | json   | workflow-start, pre-dispatch, during-execute, exit-handler | All workflow labels as a JSON object (deprecated — use workflow.labels.json)            |
| `workflow.labels.<name>`             | string | workflow-start, pre-dispatch, during-execute, exit-handler | Workflow metadata label value                                                           |
| `workflow.labels.json`               | json   | workflow-start, pre-dispatch, during-execute, exit-handler | All workflow labels as a JSON object                                                    |
| `workflow.mainEntrypoint`            | string | workflow-start, pre-dispatch, during-execute, exit-handler | spec.entrypoint                                                                         |
| `workflow.name`                      | string | workflow-start, pre-dispatch, during-execute, exit-handler | Workflow object name                                                                    |
| `workflow.namespace`                 | string | workflow-start, pre-dispatch, during-execute, exit-handler | Workflow namespace                                                                      |
| `workflow.parameters`                | json   | workflow-start, pre-dispatch, during-execute, exit-handler | All workflow parameters as a JSON array                                                 |
| `workflow.parameters.<name>`         | string | workflow-start, pre-dispatch, during-execute, exit-handler | Value from spec.arguments.parameters, ConfigMap-resolved if ValueFrom is set            |
| `workflow.parameters.json`           | json   | workflow-start, pre-dispatch, during-execute, exit-handler | All workflow parameters as a JSON array (alias for workflow.parameters)                 |
| `workflow.priority`                  | string | workflow-start, pre-dispatch, during-execute, exit-handler | Workflow priority                                                                       |
| `workflow.scheduledTime`             | string | workflow-start, pre-dispatch, during-execute, exit-handler | Scheduled time for cron-triggered workflows (from annotation)                           |
| `workflow.serviceAccountName`        | string | workflow-start, pre-dispatch, during-execute, exit-handler | Effective service account name                                                          |
| `workflow.uid`                       | string | workflow-start, pre-dispatch, during-execute, exit-handler | Workflow UID                                                                            |

### Runtime

|         Key         |  Type  |                Availability                |                              Description                              |
|---------------------|--------|--------------------------------------------|-----------------------------------------------------------------------|
| `workflow.duration` | string | pre-dispatch, during-execute, exit-handler | Elapsed seconds as float string; final at exit handler                |
| `workflow.failures` | json   | exit-handler                               | JSON array of failed node descriptors; populated when any node failed |
| `workflow.status`   | string | pre-dispatch, during-execute, exit-handler | Current workflow phase; final value only at exit handler              |

### Input

|               Key                |     Type      |  Availability  |                            Description                             |
|----------------------------------|---------------|----------------|--------------------------------------------------------------------|
| `inputs.artifacts.<name>`        | wfv1.Artifact | during-execute | Input artifact object (for fromExpression use)                     |
| `inputs.artifacts.<name>.path`   | string        | during-execute | Mount path of the input artifact inside the pod                    |
| `inputs.parameters`              | json          | during-execute | All input parameters as a JSON array                               |
| `inputs.parameters.<name>`       | string        | during-execute | Resolved input parameter value                                     |
| `outputs.artifacts.<name>.path`  | string        | during-execute | Declared output artifact path for the current template (pod side)  |
| `outputs.parameters.<name>.path` | string        | during-execute | Declared output parameter path for the current template (pod side) |

### Node-Ref

|                    Key                    |     Type      |         Availability         |                              Description                              |
|-------------------------------------------|---------------|------------------------------|-----------------------------------------------------------------------|
| `steps.<loopName>.outputs.parameters`     | json          | after-loop                   | JSON array of per-child output-parameter maps                         |
| `steps.<loopName>.outputs.parameters.<p>` | json          | after-loop                   | JSON array of values for a named parameter across all children        |
| `steps.<loopName>.outputs.result`         | json          | after-loop                   | JSON array of child results (withItems/withParam)                     |
| `steps.<name>.exitCode`                   | string        | after-node-complete          | Container exit code                                                   |
| `steps.<name>.finishedAt`                 | string        | after-node-complete          | RFC3339 finish time                                                   |
| `steps.<name>.hostNodeName`               | string        | after-pod-start              | Underlying k8s node name                                              |
| `steps.<name>.id`                         | string        | after-node-init              | Node ID                                                               |
| `steps.<name>.ip`                         | string        | after-pod-start              | Pod IP                                                                |
| `steps.<name>.outputs.artifacts.<a>`      | wfv1.Artifact | after-node-succeeded         | Named output artifact of the referenced node                          |
| `steps.<name>.outputs.parameters.<p>`     | string        | after-node-succeeded         | Named output parameter of the referenced node                         |
| `steps.<name>.outputs.result`             | string        | after-node-succeeded         | Captured stdout (non-loop nodes)                                      |
| `steps.<name>.startedAt`                  | string        | after-pod-start              | RFC3339 start time                                                    |
| `steps.<name>.status`                     | string        | after-node-init              | Node phase                                                            |
| `tasks.<loopName>.outputs.parameters`     | json          | after-loop                   | JSON array of per-child output-parameter maps                         |
| `tasks.<loopName>.outputs.parameters.<p>` | json          | after-loop                   | JSON array of values for a named parameter across all children        |
| `tasks.<loopName>.outputs.result`         | json          | after-loop                   | JSON array of child results (withItems/withParam)                     |
| `tasks.<name>.exitCode`                   | string        | after-node-complete          | Container exit code                                                   |
| `tasks.<name>.finishedAt`                 | string        | after-node-complete          | RFC3339 finish time                                                   |
| `tasks.<name>.hostNodeName`               | string        | after-pod-start              | Underlying k8s node name                                              |
| `tasks.<name>.id`                         | string        | after-node-init              | Node ID                                                               |
| `tasks.<name>.ip`                         | string        | after-pod-start              | Pod IP                                                                |
| `tasks.<name>.outputs.artifacts.<a>`      | wfv1.Artifact | after-node-succeeded         | Named output artifact of the referenced node                          |
| `tasks.<name>.outputs.parameters.<p>`     | string        | after-node-succeeded         | Named output parameter of the referenced node                         |
| `tasks.<name>.outputs.result`             | string        | after-node-succeeded         | Captured stdout (non-loop nodes)                                      |
| `tasks.<name>.startedAt`                  | string        | after-pod-start              | RFC3339 start time                                                    |
| `tasks.<name>.status`                     | string        | after-node-init              | Node phase                                                            |
| `workflow.outputs.artifacts.<name>`       | wfv1.Artifact | during-execute, exit-handler | Global output artifact (lifted via outputs.artifacts[*].globalName)   |
| `workflow.outputs.parameters.<name>`      | string        | during-execute, exit-handler | Global output parameter (lifted via outputs.parameters[*].globalName) |

### Item

|     Key      |    Type     |        Availability         |                                 Description                                  |
|--------------|-------------|-----------------------------|------------------------------------------------------------------------------|
| `item`       | string|json | inside-loop, during-execute | Current loop iteration value (withItems/withParam). JSON for map/list items. |
| `item.<key>` | string      | inside-loop, during-execute | Accessor into a map-typed loop iteration value                               |

### Retry

|          Key           |  Type  |         Availability         |                        Description                        |
|------------------------|--------|------------------------------|-----------------------------------------------------------|
| `retries`              | string | inside-retry, during-execute | 0-based retry attempt index                               |
| `retries.lastDuration` | string | inside-retry, during-execute | Duration of the previous attempt in seconds               |
| `retries.lastExitCode` | string | inside-retry, during-execute | Exit code of the previous attempt (or 0 on first attempt) |
| `retries.lastMessage`  | string | inside-retry, during-execute | Message of the previous attempt                           |
| `retries.lastStatus`   | string | inside-retry, during-execute | Phase of the previous attempt (or empty on first)         |

### Node-Ctx

|     Key      |  Type  |         Availability         |                       Description                       |
|--------------|--------|------------------------------|---------------------------------------------------------|
| `node.name`  | string | pre-dispatch, during-execute | Full node name                                          |
| `pod.name`   | string | pre-dispatch, during-execute | Computed pod name for pod-producing templates           |
| `steps.name` | string | pre-dispatch, during-execute | Name of the current step (inside a Steps template body) |
| `tasks.name` | string | pre-dispatch, during-execute | Name of the current task (inside a DAG template body)   |

## 3. Matrix by TemplateKind

Which variables are in scope for each template type. `•` = in scope, blank = not in scope.

|                    Key                    | any | container | script | resource | steps | dag | data | suspend | http | plugin | exit-handler |
|-------------------------------------------|-----|-----------|--------|----------|-------|-----|------|---------|------|--------|--------------|
| `inputs.artifacts.<name>`                 | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `inputs.artifacts.<name>.path`            |     | •         | •      | •        |       |     |      |         |      |        |              |
| `inputs.parameters`                       | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `inputs.parameters.<name>`                | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `item`                                    | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `item.<key>`                              | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `node.name`                               | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `outputs.artifacts.<name>.path`           |     | •         | •      | •        |       |     |      |         |      |        |              |
| `outputs.parameters.<name>.path`          |     | •         | •      | •        |       |     |      |         |      |        |              |
| `pod.name`                                |     | •         | •      | •        |       |     |      |         |      |        |              |
| `retries`                                 |     | •         | •      | •        |       |     |      |         |      |        |              |
| `retries.lastDuration`                    |     | •         | •      | •        |       |     |      |         |      |        |              |
| `retries.lastExitCode`                    |     | •         | •      | •        |       |     |      |         |      |        |              |
| `retries.lastMessage`                     |     | •         | •      | •        |       |     |      |         |      |        |              |
| `retries.lastStatus`                      |     | •         | •      | •        |       |     |      |         |      |        |              |
| `steps.<loopName>.outputs.parameters`     |     |           |        |          | •     |     |      |         |      |        | •            |
| `steps.<loopName>.outputs.parameters.<p>` |     |           |        |          | •     |     |      |         |      |        | •            |
| `steps.<loopName>.outputs.result`         |     |           |        |          | •     |     |      |         |      |        | •            |
| `steps.<name>.exitCode`                   |     |           |        |          | •     |     |      |         |      |        | •            |
| `steps.<name>.finishedAt`                 |     |           |        |          | •     |     |      |         |      |        | •            |
| `steps.<name>.hostNodeName`               |     |           |        |          | •     |     |      |         |      |        | •            |
| `steps.<name>.id`                         |     |           |        |          | •     |     |      |         |      |        | •            |
| `steps.<name>.ip`                         |     |           |        |          | •     |     |      |         |      |        | •            |
| `steps.<name>.outputs.artifacts.<a>`      |     |           |        |          | •     |     |      |         |      |        | •            |
| `steps.<name>.outputs.parameters.<p>`     |     |           |        |          | •     |     |      |         |      |        | •            |
| `steps.<name>.outputs.result`             |     |           |        |          | •     |     |      |         |      |        | •            |
| `steps.<name>.startedAt`                  |     |           |        |          | •     |     |      |         |      |        | •            |
| `steps.<name>.status`                     |     |           |        |          | •     |     |      |         |      |        | •            |
| `steps.name`                              |     |           |        |          | •     |     |      |         |      |        |              |
| `tasks.<loopName>.outputs.parameters`     |     |           |        |          |       | •   |      |         |      |        | •            |
| `tasks.<loopName>.outputs.parameters.<p>` |     |           |        |          |       | •   |      |         |      |        | •            |
| `tasks.<loopName>.outputs.result`         |     |           |        |          |       | •   |      |         |      |        | •            |
| `tasks.<name>.exitCode`                   |     |           |        |          |       | •   |      |         |      |        | •            |
| `tasks.<name>.finishedAt`                 |     |           |        |          |       | •   |      |         |      |        | •            |
| `tasks.<name>.hostNodeName`               |     |           |        |          |       | •   |      |         |      |        | •            |
| `tasks.<name>.id`                         |     |           |        |          |       | •   |      |         |      |        | •            |
| `tasks.<name>.ip`                         |     |           |        |          |       | •   |      |         |      |        | •            |
| `tasks.<name>.outputs.artifacts.<a>`      |     |           |        |          |       | •   |      |         |      |        | •            |
| `tasks.<name>.outputs.parameters.<p>`     |     |           |        |          |       | •   |      |         |      |        | •            |
| `tasks.<name>.outputs.result`             |     |           |        |          |       | •   |      |         |      |        | •            |
| `tasks.<name>.startedAt`                  |     |           |        |          |       | •   |      |         |      |        | •            |
| `tasks.<name>.status`                     |     |           |        |          |       | •   |      |         |      |        | •            |
| `tasks.name`                              |     |           |        |          |       | •   |      |         |      |        |              |
| `workflow.annotations`                    | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.annotations.<name>`             | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.annotations.json`               | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.creationTimestamp`              | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.creationTimestamp.<fmt>`        | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.creationTimestamp.RFC3339`      | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.creationTimestamp.s`            | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.duration`                       | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.failures`                       | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.labels`                         | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.labels.<name>`                  | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.labels.json`                    | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.mainEntrypoint`                 | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.name`                           | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.namespace`                      | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.outputs.artifacts.<name>`       | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.outputs.parameters.<name>`      | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.parameters`                     | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.parameters.<name>`              | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.parameters.json`                | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.priority`                       | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.scheduledTime`                  | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.serviceAccountName`             | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.status`                         | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |
| `workflow.uid`                            | •   | •         | •      | •        | •     | •   | •    | •       | •    | •      | •            |

## 4. Grouped by LifecyclePhase

|        Phase         |                                                                    Meaning                                                                    |
|----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------|
| workflow-start       | Globals populated once, up front, before any template runs.                                                                                   |
| pre-dispatch         | Immediately before a template's pod is created; pod.name / node.name / steps.name / tasks.name are set.                                       |
| during-execute       | Inside a template body; inputs.* are bound.                                                                                                   |
| inside-loop          | Inside a withItems/withParam expansion; `item`, `item.<key>` are bound.                                                                       |
| inside-retry         | Inside a retryStrategy template; retries.* are bound.                                                                                         |
| after-node-init      | A referenced node has been initialised (has an ID / phase). Earliest steps.X.id, steps.X.status.                                              |
| after-pod-start      | The referenced node's pod has started; startedAt, ip, hostNodeName are populated.                                                             |
| after-node-complete  | The referenced node has finished (any terminal phase); finishedAt, exitCode are populated.                                                    |
| after-node-succeeded | The referenced node has finished with Succeeded; outputs.result, outputs.parameters.*, outputs.artifacts.* are populated.                     |
| after-loop           | Every child of a withItems/withParam group has completed; aggregated outputs appear.                                                          |
| exit-handler         | The onExit template runs. workflow.{status,failures,duration} are final. Any earlier-phase variable is also visible here (scope accumulates). |


### workflow-start (20 variables)

|                 Key                  |  Kind  |  Type  |
|--------------------------------------|--------|--------|
| `workflow.annotations`               | global | json   |
| `workflow.annotations.<name>`        | global | string |
| `workflow.annotations.json`          | global | json   |
| `workflow.creationTimestamp`         | global | string |
| `workflow.creationTimestamp.<fmt>`   | global | string |
| `workflow.creationTimestamp.RFC3339` | global | string |
| `workflow.creationTimestamp.s`       | global | string |
| `workflow.labels`                    | global | json   |
| `workflow.labels.<name>`             | global | string |
| `workflow.labels.json`               | global | json   |
| `workflow.mainEntrypoint`            | global | string |
| `workflow.name`                      | global | string |
| `workflow.namespace`                 | global | string |
| `workflow.parameters`                | global | json   |
| `workflow.parameters.<name>`         | global | string |
| `workflow.parameters.json`           | global | json   |
| `workflow.priority`                  | global | string |
| `workflow.scheduledTime`             | global | string |
| `workflow.serviceAccountName`        | global | string |
| `workflow.uid`                       | global | string |

### pre-dispatch (26 variables)

|                 Key                  |   Kind   |  Type  |
|--------------------------------------|----------|--------|
| `node.name`                          | node-ctx | string |
| `pod.name`                           | node-ctx | string |
| `steps.name`                         | node-ctx | string |
| `tasks.name`                         | node-ctx | string |
| `workflow.annotations`               | global   | json   |
| `workflow.annotations.<name>`        | global   | string |
| `workflow.annotations.json`          | global   | json   |
| `workflow.creationTimestamp`         | global   | string |
| `workflow.creationTimestamp.<fmt>`   | global   | string |
| `workflow.creationTimestamp.RFC3339` | global   | string |
| `workflow.creationTimestamp.s`       | global   | string |
| `workflow.duration`                  | runtime  | string |
| `workflow.labels`                    | global   | json   |
| `workflow.labels.<name>`             | global   | string |
| `workflow.labels.json`               | global   | json   |
| `workflow.mainEntrypoint`            | global   | string |
| `workflow.name`                      | global   | string |
| `workflow.namespace`                 | global   | string |
| `workflow.parameters`                | global   | json   |
| `workflow.parameters.<name>`         | global   | string |
| `workflow.parameters.json`           | global   | json   |
| `workflow.priority`                  | global   | string |
| `workflow.scheduledTime`             | global   | string |
| `workflow.serviceAccountName`        | global   | string |
| `workflow.status`                    | runtime  | string |
| `workflow.uid`                       | global   | string |

### during-execute (41 variables)

|                 Key                  |   Kind   |     Type      |
|--------------------------------------|----------|---------------|
| `inputs.artifacts.<name>`            | input    | wfv1.Artifact |
| `inputs.artifacts.<name>.path`       | input    | string        |
| `inputs.parameters`                  | input    | json          |
| `inputs.parameters.<name>`           | input    | string        |
| `item`                               | item     | string|json   |
| `item.<key>`                         | item     | string        |
| `node.name`                          | node-ctx | string        |
| `outputs.artifacts.<name>.path`      | input    | string        |
| `outputs.parameters.<name>.path`     | input    | string        |
| `pod.name`                           | node-ctx | string        |
| `retries`                            | retry    | string        |
| `retries.lastDuration`               | retry    | string        |
| `retries.lastExitCode`               | retry    | string        |
| `retries.lastMessage`                | retry    | string        |
| `retries.lastStatus`                 | retry    | string        |
| `steps.name`                         | node-ctx | string        |
| `tasks.name`                         | node-ctx | string        |
| `workflow.annotations`               | global   | json          |
| `workflow.annotations.<name>`        | global   | string        |
| `workflow.annotations.json`          | global   | json          |
| `workflow.creationTimestamp`         | global   | string        |
| `workflow.creationTimestamp.<fmt>`   | global   | string        |
| `workflow.creationTimestamp.RFC3339` | global   | string        |
| `workflow.creationTimestamp.s`       | global   | string        |
| `workflow.duration`                  | runtime  | string        |
| `workflow.labels`                    | global   | json          |
| `workflow.labels.<name>`             | global   | string        |
| `workflow.labels.json`               | global   | json          |
| `workflow.mainEntrypoint`            | global   | string        |
| `workflow.name`                      | global   | string        |
| `workflow.namespace`                 | global   | string        |
| `workflow.outputs.artifacts.<name>`  | node-ref | wfv1.Artifact |
| `workflow.outputs.parameters.<name>` | node-ref | string        |
| `workflow.parameters`                | global   | json          |
| `workflow.parameters.<name>`         | global   | string        |
| `workflow.parameters.json`           | global   | json          |
| `workflow.priority`                  | global   | string        |
| `workflow.scheduledTime`             | global   | string        |
| `workflow.serviceAccountName`        | global   | string        |
| `workflow.status`                    | runtime  | string        |
| `workflow.uid`                       | global   | string        |

### inside-loop (2 variables)

|     Key      | Kind |    Type     |
|--------------|------|-------------|
| `item`       | item | string|json |
| `item.<key>` | item | string      |

### inside-retry (5 variables)

|          Key           | Kind  |  Type  |
|------------------------|-------|--------|
| `retries`              | retry | string |
| `retries.lastDuration` | retry | string |
| `retries.lastExitCode` | retry | string |
| `retries.lastMessage`  | retry | string |
| `retries.lastStatus`   | retry | string |

### after-node-init (4 variables)

|          Key          |   Kind   |  Type  |
|-----------------------|----------|--------|
| `steps.<name>.id`     | node-ref | string |
| `steps.<name>.status` | node-ref | string |
| `tasks.<name>.id`     | node-ref | string |
| `tasks.<name>.status` | node-ref | string |

### after-pod-start (6 variables)

|             Key             |   Kind   |  Type  |
|-----------------------------|----------|--------|
| `steps.<name>.hostNodeName` | node-ref | string |
| `steps.<name>.ip`           | node-ref | string |
| `steps.<name>.startedAt`    | node-ref | string |
| `tasks.<name>.hostNodeName` | node-ref | string |
| `tasks.<name>.ip`           | node-ref | string |
| `tasks.<name>.startedAt`    | node-ref | string |

### after-node-complete (4 variables)

|            Key            |   Kind   |  Type  |
|---------------------------|----------|--------|
| `steps.<name>.exitCode`   | node-ref | string |
| `steps.<name>.finishedAt` | node-ref | string |
| `tasks.<name>.exitCode`   | node-ref | string |
| `tasks.<name>.finishedAt` | node-ref | string |

### after-node-succeeded (6 variables)

|                  Key                  |   Kind   |     Type      |
|---------------------------------------|----------|---------------|
| `steps.<name>.outputs.artifacts.<a>`  | node-ref | wfv1.Artifact |
| `steps.<name>.outputs.parameters.<p>` | node-ref | string        |
| `steps.<name>.outputs.result`         | node-ref | string        |
| `tasks.<name>.outputs.artifacts.<a>`  | node-ref | wfv1.Artifact |
| `tasks.<name>.outputs.parameters.<p>` | node-ref | string        |
| `tasks.<name>.outputs.result`         | node-ref | string        |

### after-loop (6 variables)

|                    Key                    |   Kind   | Type |
|-------------------------------------------|----------|------|
| `steps.<loopName>.outputs.parameters`     | node-ref | json |
| `steps.<loopName>.outputs.parameters.<p>` | node-ref | json |
| `steps.<loopName>.outputs.result`         | node-ref | json |
| `tasks.<loopName>.outputs.parameters`     | node-ref | json |
| `tasks.<loopName>.outputs.parameters.<p>` | node-ref | json |
| `tasks.<loopName>.outputs.result`         | node-ref | json |

### exit-handler (25 variables)

|                 Key                  |   Kind   |     Type      |
|--------------------------------------|----------|---------------|
| `workflow.annotations`               | global   | json          |
| `workflow.annotations.<name>`        | global   | string        |
| `workflow.annotations.json`          | global   | json          |
| `workflow.creationTimestamp`         | global   | string        |
| `workflow.creationTimestamp.<fmt>`   | global   | string        |
| `workflow.creationTimestamp.RFC3339` | global   | string        |
| `workflow.creationTimestamp.s`       | global   | string        |
| `workflow.duration`                  | runtime  | string        |
| `workflow.failures`                  | runtime  | json          |
| `workflow.labels`                    | global   | json          |
| `workflow.labels.<name>`             | global   | string        |
| `workflow.labels.json`               | global   | json          |
| `workflow.mainEntrypoint`            | global   | string        |
| `workflow.name`                      | global   | string        |
| `workflow.namespace`                 | global   | string        |
| `workflow.outputs.artifacts.<name>`  | node-ref | wfv1.Artifact |
| `workflow.outputs.parameters.<name>` | node-ref | string        |
| `workflow.parameters`                | global   | json          |
| `workflow.parameters.<name>`         | global   | string        |
| `workflow.parameters.json`           | global   | json          |
| `workflow.priority`                  | global   | string        |
| `workflow.scheduledTime`             | global   | string        |
| `workflow.serviceAccountName`        | global   | string        |
| `workflow.status`                    | runtime  | string        |
| `workflow.uid`                       | global   | string        |
