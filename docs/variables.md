# Workflow Variables

The following variables are made available to reference various metadata of a workflow:

## All Templates:
| Variable | Description|
|----------|------------|
| `inputs.parameters.<NAME>`| Input parameter to a template |
| `inputs.artifacts.<NAME>` | Input artifact to a template |

## Steps Templates:
| Variable | Description|
|----------|------------|
| `steps.<STEPNAME>.ip` | IP address of a previous daemon container step |
| `steps.<STEPNAME>.outputs.result` | Output result of a previous script step |
| `steps.<STEPNAME>.outputs.parameters.<NAME>` | Output parameter of a previous step |
| `steps.<STEPNAME>.outputs.artifacts.<NAME>` | Output artifact of a previous step |

## DAG Templates:
| Variable | Description|
|----------|------------|
| `tasks.<TASKNAME>.ip` | IP address of a previous daemon container task |
| `tasks.<TASKNAME>.outputs.result` | Output result of a previous script task |
| `tasks.<TASKNAME>.outputs.parameters.<NAME>` | Output parameter of a previous task |
| `tasks.<TASKNAME>.outputs.artifacts.<NAME>` | Output artifact of a previous task |

## Container/Script Templates:
| Variable | Description|
|----------|------------|
| `pod.name` | Pod name of the container/script |
| `inputs.artifacts.<NAME>.path` | Local path of the input artifact |
| `outputs.artifacts.<NAME>.path` | Local path of the output artifact |
| `outputs.parameters.<NAME>.path` | Local path of the output parameter |

## Loops (withItems / withParam)
| Variable | Description|
|----------|------------|
| `item` | Value of the item in a list |
| `item.<FIELDNAME>` | Field value of the item in a list of maps |

## Global:
| Variable | Description|
|----------|------------|
| `workflow.name` | Workflow name |
| `workflow.namespace` | Workflow namespace |
| `workflow.uid` | Workflow UID. Useful for setting ownership reference to a resource, or a unique artifact location |
| `workflow.parameters.<NAME>` | Input parameter to the workflow |
| `workflow.outputs.parameters.<NAME>` | Input artifact to the workflow |
| `workflow.annotations.<NAME>` | Workflow annotations |
| `workflow.labels.<NAME>` | Workflow labels |
| `workflow.creationTimestamp` | Workflow creation timestamp formatted in RFC 3339  (e.g. `2018-08-23T05:42:49Z`) |
| `workflow.creationTimestamp.<STRFTIMECHAR>` | Creation timestamp formatted with a [strftime](http://strftime.org) format character |


## Exit Handler:
| Variable | Description|
|----------|------------|
| `workflow.status` | Workflow status. One of: `Succeeded`, `Failed`, `Error` |
