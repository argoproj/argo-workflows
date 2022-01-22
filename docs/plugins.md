# Plugins

![Plugins](assets/plugins.png)

Plugins allow you to extend Argo Workflows to add new capabilities.

* You don't need to learn Golang, you can write in any language, including Python.
* Simple: a plugin just responds to RPC HTTP requests.
* You can iterate quickly by changing the plugin at runtime.
* You can get your plugin running today, no need to wait 3-5 months for review, approval, merge and an Argo software
  release.

[Executor plugins](executor_plugins.md) can be written and installed by both users and admins.

## Configuration

Plugins are disabled by default. To enable them, start the controller with `ARGO_EXECUTOR_PLUGINS=true`, e.g.

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
            - name: ARGO_EXECUTOR_PLUGINS
              value: "true"
```

## Failures

A plugin may fail as follows:

* Connection/socket error - considered transient.
* Timeout - considered transient.
* 404 error - method is not supported by the plugin, as a result the method will not be called again.
* 503 error - considered transient.
* Other 4xx/5xx errors - considered fatal.

Transient errors are retried, all other errors are considered fatal.

Fatal errors will result in failed steps.

