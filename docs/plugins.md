# Plugins

![Plugins](assets/plugins.png)

Plugins allow you to extend Argo Workflows to add new capabilities.

* You don't need to learn Golang, you can write in any language, including Python.
* Simple: a plugin just responds to RPC HTTP requests.
* You can iterate quickly by changing the plugin at runtime.
* You can get your plugin running today, no need to wait 3-5 months for an Argo software release.

There are two types of plugins

* [Executor plugins](executor_plugins.md) written and installed by both users and admins.
* [Controller plugins](controller_plugins.md) written and installed only by the admin.

## Configuration

Plugins are disabled by default. To enable them, start the controller with `ARGO_PLUGINS=true`, e.g.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: workflow-controller
spec:
  template:
    spec:
      containers:
        - name: workflow-controller
          env:
            - name: ARGO_PLUGINS
              value: "true"
```
