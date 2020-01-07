# Offloading Large Workflows

![beta](assets/beta.svg)

Argo stores workflows as Kubernetes resources (i.e. within EtcD). This creates a limit to their size as resources must be under 1MB. Each resource includes the status of each node, which is stored in the `/status/nodes` field for the resource. This can be over 1MB. If this happens, we try and compress the node status and store it in `/status/compressedNodes`. If the status is still too large, we then try and store it in an SQL database. 

To enable this feature, configure a Postgres or MySQL database under `persistence` in [your configuration](workflow-controller-configmap.yaml) and set `nodeStatusOffLoad: true`.