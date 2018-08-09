# Workflow Variables

The following variables are made available to reference various metadata of a workflow:

## Templates:
* `inputs.artifacts.XXX`
* `inputs.parameters.XXX`

## Steps Templates:
* `steps.XXX.ip`
* `steps.XXX.outputs.result`
* `steps.XXX.outputs.parameters.YYY`
* `steps.XXX.outputs.artifacts.YYY`

## DAG Templates:
* `tasks.XXX.ip`
* `tasks.XXX.outputs.result`
* `tasks.XXX.outputs.parameters.YYY`
* `tasks.XXX.outputs.artifacts.YYY`

## Container/Script/Resource Templates:
* `pod.name`

## Loops
* `item`
* `item.XXX`

## Global:
* `workflow.name`
* `workflow.namespace`
* `workflow.uid`
* `workflow.parameters.XXX`
* `workflow.outputs.parameters.XXX`

## Exit Handler:
* `workflow.status`

## Coming in v2.2:
* `workflow.artifacts.XXX`
* `workflow.outputs.artifacts.XXX`
