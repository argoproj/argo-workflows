Description: Add labels to help tracking workflow of workflows
Author: [Chengwei Guo](https://github.com/cw-Guo)
Component: General
Issues: 6922

When creating the resource in resouce template, attach some labels to track the parent workflow name and parent node id. Such labels can be used to create custom links to jump between parent workflow and children workflow. This is a work-around solution for the Issue 6922.
It only works when `setOwnerReference` is set to True.

Example:
```yaml
Links:
  - name: Child Workflow
    scope: pod
    url: http://localhost:8080/workflows/${metadata.namespace}?label=workflows.argoproj.io/resource-parent-pod-name=${metadata.name}
  - name: Parent Workflow
    scope: workflow
    url: http://localhost:8080/workflows/${metadata.namespace}/${workflow.metadata.labels.workflows.argoproj.io/resource-parent-workflow-name}
  - name: Child Workflows
    scope: workflow
    url: http://localhost:8080/workflows/${metadata.namespace}?label=workflows.argoproj.io/resource-parent-workflow-name=${metadata.name}
```