Description: Keep created Workflows when deleting a CronWorkflow in the UI
Authors: [JerryNee](https://github.com/JerryNee)
Component: UI
Issues: 14679

The CronWorkflow deletion confirmation now includes an option to keep its non-archived Workflows.
Selecting the option deletes the CronWorkflow with Kubernetes orphan propagation so the created Workflows remain available.
