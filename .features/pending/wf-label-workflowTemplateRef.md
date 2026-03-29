Description: Add WorkflowTemplate name as label when using workflowTemplateRef
Authors: [Eduardo Rodrigues](https://github.com/eduardodbr)
Component: General
Issues: 12670

When a `Workflow` or a `CronWorkflow` is submitted from a `WorkflowTemplate` or `ClusterWorkflowTemplate` ( i.e. using the `workflowTemplateRef`) it stores the the `WorkflowTemplate` name as a label.