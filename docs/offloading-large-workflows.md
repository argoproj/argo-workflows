# Offloading Large Workflows

> v2.4 and after

Argo stores workflows as Kubernetes resources (i.e. within EtcD). This creates a limit to their size as resources must be under 1MB. Each resource includes the status of each node, which is stored in the `/status/nodes` field for the resource. This can be over 1MB. If this happens, we try and compress the node status and store it in `/status/compressedNodes`. If the status is still too large, we then try and store it in an SQL database.

To enable this feature, configure a Postgres or MySQL database under `persistence` in [your configuration](workflow-controller-configmap.yaml) and set `nodeStatusOffLoad: true`.

## FAQ

### Why aren't my workflows appearing in the database?

Offloading is expensive and often unnecessary, so we only offload when we need to. Your workflows aren't probably large enough.

### Error `Failed to submit workflow: etcdserver: request is too large.`

You must use the Argo CLI having exported `export ARGO_SERVER=...`.

### Error `offload node status is not supported`

Even after compressing node statuses, the workflow exceeded the EtcD
size limit. To resolve, either enable node status offload as described
above or look for ways to reduce the size of your workflow manifest:

- Use `withItems` or `withParams` to consolidate similar templates into a single parametrized template
- Use [template defaults](template-defaults.md) to factor shared template options to the workflow level
- Use [workflow templates](workflow-templates.md) to factor frequently-used templates into separate resources
- Use [workflows of workflows](workflow-of-workflows.md) to factor a large workflow into a workflow of smaller workflows
