# Node Field Selectors

![GA](assets/ga.svg)

> v2.8 and after

## Introduction

The resume, stop and retry Argo CLI and API commands support a `--node-field-selector` parameter to allow the user to select a subset of nodes for the command to apply to. 

In the case of the resume and stop commands these are the nodes that should be resumed or stopped.

In the case of the retry command it allows specifying nodes that should be restarted even if they were previously successful (and must be used in combination with `--restart-successful`)

The format of this when used with the CLI is:

```--node-field-selector=FIELD=VALUE```

## Possible options

The field can be any of:

| Field | Description|
|----------|------------|
| displayName | Display name of the node |
| templateName | Template name of the node |
| phase | Phase status of the node - eg Running |
| templateRef.name | The name of the WorkflowTemplate the node is referring to |
| templateRef.template | The template within the WorkflowTemplate the node is referring to |
| inputs.parameters.<NAME>.value | The value of input parameter NAME |

The operator can be '=' or '!='. Multiple selectors can be combined with a comma, in which case they are ANDed together.

## Examples

To filter for nodes where the input parameter 'foo' is equal to 'bar':

```--node-field-selector=inputs.parameters.foo.value=bar```

To filter for nodes where the input parameter 'foo' is equal to 'bar' and phase is not running:

```--node-field-selector=foo1=bar1,phase!=Running```
