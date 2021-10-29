# Plugins

Plugins allow you to extend Argo Workflows.

## Workflow Lifecycle Hook

* Modify a workflow before it is operated, e.g. to validate it.
* Record meta-data and lineage.
* Real-time progress reporting.
* Emit notifications.

## Template Executor

* Run custom templates.
* Offload caching decision to an external system.
* Allow extra placeholders using data from an external system.
* Block workflow execution until an external system has finished logging some metadata for one task.
* Emit notification.
* Support operators, such as Spark.

## Writing A Plugin

Plugins are Go plugins:

* Only Linux and MacOS, now Windows.
* You must re-build them for each new version of Argo Workflows.

Look at the following example workflow/controller/plugins/hello/plugin.go.

Why not an RPC sidecar plugin?

* You can build RPC sidecar as a Go plugin, but not vice-versa.
* O(ab) network calls: num(nodes) x num(plugins)
