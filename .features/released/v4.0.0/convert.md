Description: `convert` CLI command to convert to new workflow format
Authors: [Alan Clucas](https://github.com/Joibel)
Component: CLI
Issues: 14977

A new CLI command `convert` which will convert Workflows, CronWorkflows, and (Cluster)WorkflowTemplates to the new format.
It will remove `schedule` from CronWorkflows, moving that into `schedules`
It will remove `mutex` and `semaphore` from `synchronization` blocks and move them to the plural version.
Otherwise this command works much the same as linting.
