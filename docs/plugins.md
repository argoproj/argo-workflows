# Executor Plugins

![Plugins](assets/plugins.png)

Executor Plugins extend Argo Workflows by adding new capabilities without requiring changes to the Argo Workflows codebase.
This allows you to iterate quickly and add new features without waiting for a new release.

Executor Plugins are containerized applications that respond to RPC HTTP requests.
You can write Executor Plugins in any language, so you don't need to learn Golang to extend Argo Workflows.

When invoked, Executor Plugins run in a special `agent` pod that the Argo Workflows controller creates and manages.
Each running workflow uses only one Agent pod, which can improve performance when running multiple steps that use the same plugin.

The same Agent pod also runs any [HTTP templates](http-template.md) that are part of the Workflow, offering additional performance advantages.

You define [Executor plugin configuration](executor_plugins.md) as an `ExecutorPlugin` CustomResource. Both users and admins can write and install them.
