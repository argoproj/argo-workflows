Description: Add Optional Argo Workflow–Level Configuration for Executor Plugins
Author: [ntny](https://github.com/ntny)
Component: General
Issues: 15234

This PR allows configuring the Argo Workflow Executor Plugin for a specific Argo Workflow directly within the Workflow spec.
Enable this with the `ARGO_WORKFLOW_LEVEL_EXECUTOR_PLUGINS=true` controller environment variable.
Workflow-level executor plugin settings take precedence over globally configured executor plugins.

See `docs/executor_plugins.md` for configuration details and examples.
