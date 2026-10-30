Description: View a workflow pod's specification and status live from the node-info panel in the UI
Authors: [panicboat](https://github.com/panicboat)
Component: UI
Issues: 14538

The node-info panel of a Pod node has a new POD button.
It opens the Kubernetes Pod of that node as YAML or JSON and keeps it up to date while the panel is open.
Use it to find out why a Pod is stuck in `Pending` or failed to start: the requested resources, node selector and affinity, volumes, security context, conditions and container statuses are all in the Pod.
The panel reads the Pod through a new `WatchWorkflowPod` API (`GET /api/v1/stream/workflows/{namespace}/{name}/pods/{podName}`), which needs permission to read the Workflow and to watch Pods in its namespace.
The button doesn't appear for workflows that exist only in the archive database, and you can't view a Pod that has already been deleted.
