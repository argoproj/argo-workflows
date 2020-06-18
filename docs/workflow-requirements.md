# Workflow Requirements

![alpha](assets/alpha.svg)

> v2.9 and after

## Introduction

As the administrator of the controller, you may want to limit which types of Workflows your users can run. Setting Workflow requirements allows you to ensure that Workflows comply with certain requirements. 

## Available Requirements

* `referenceOnly`: Only Workflows using "workflowTemplateRef" will be processed. This allows the administrator of the controller to set a "library" of templates that may be run by its opeartor, limiting arbitrary Workflow execution.
* `strictReferenceOnly`: Only Workflows using "workflowTemplateRef" will be processed and the controller will enforce that the WorkflowTemplate that is referenced hasn't changed between operations. If you want to make sure the operator of the Workflow cannot run an arbitrary Workflow, use this option.

## Setting Execution Requirements

Workflow Requirements can be specified by adding them under the `workflowRequirements` key in the [`workflow-controller-configmap`](./workflow-controller-configmap.yaml).

For example, to specify that Workflows may only run with `workflowTemplateRef`

```yaml
# This file describes the config settings available in the workflow controller configmap
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data: |
  workflowRequirements:
    referenceOnly: true

```
