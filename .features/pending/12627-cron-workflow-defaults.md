Component: CronWorkflows
Issues: 12627
Description: Add `cronWorkflowDefaults` to the controller ConfigMap so default `CronWorkflow` spec values can be set controller-wide
Author: [alien2003](https://github.com/alien2003)

`cronWorkflowDefaults` mirrors the existing `workflowDefaults`, but for `CronWorkflow` spec fields such as `concurrencyPolicy`, `startingDeadlineSeconds`, `successfulJobsHistoryLimit`, `failedJobsHistoryLimit` and `timezone`.
Use it to enforce settings across every `CronWorkflow` a controller manages, for example capping the number of retained child Workflows for cost control without editing each `CronWorkflow`.
A value set on the `CronWorkflow` itself always takes precedence over the default.
Defaults are applied to the controller's in-memory copy during reconciliation and are never written back to the resource, so they do not conflict with GitOps tools that own the `CronWorkflow` manifest.
See [Default Workflow and CronWorkflow specs](https://argo-workflows.readthedocs.io/en/latest/default-workflow-specs/) for a configuration example.
