Description: Keep created Workflows when deleting a CronWorkflow in the UI
Authors: [JerryNee](https://github.com/JerryNee)
Component: UI
Issues: 14679

The CronWorkflow deletion confirmation in the UI now includes a **Keep Workflows created by this CronWorkflow** checkbox.
Select it when you want to stop future scheduled runs while retaining existing non-archived Workflows for inspection or history.
The UI uses Kubernetes orphan propagation so the created Workflows remain available after the CronWorkflow is deleted.
This option is available only in the UI; `argo cron delete` continues to delete the created Workflows.
