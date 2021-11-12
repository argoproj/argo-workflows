# Plugins

![Plugins](assets/plugins.png)

Plugins allow you to extend Argo Workflows to add new capabilities.

* You don't need to learn Golang, you can write in any language, including Python.
* Simple: a plugin just responds to RPC HTTP requests.
* You can iterate quickly by changing the plugin at runtime.

## Controller Plugins

### Types

#### Workflow Lifecycle Hook

View or modify a workflow before or after in is operated on.

Use cases:

* Validate it, e.g. to enforce so custom constraint.
* Record meta-data and lineage to external system.
* Real-time progress reporting.
* Notify users that a workflow completed or errored out.

#### Node Lifecycle Hook

View or modify a node before or after it is executed, including writing your own custom templates.

Use cases:

* Short-circuit execution, marking steps as complete based on external information.
* Run custom ("plugin") templates, e.g for non-pod tasks such as Tekton builds or Spark jobs.
* Offload caching decision to an external system.
* Notify users when a particular step completes.

#### Parameters Substitution Plugin

* Allow extra placeholders using data from an external system.

### Configuration

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

### A Simple Python Plugin

Lets make a Python plugin that prints "hello" each time the workflow is operated on.

We need the following:

1. A HTTP server that responds to RPC HTTP requests from the controller.
2. Configuration so the controller can discover the plugin.
3. Configuration to enable controller plugins.

We'll need to create a script that starts a HTTP server:

```python
import json
from http.server import BaseHTTPRequestHandler, HTTPServer


class Plugin(BaseHTTPRequestHandler):

    def do_POST(self):
        if self.path == "/workflow.preOperate":
            print("hello")
            self.send_response(200)
            self.end_headers()
            self.wfile.write(json.dumps({}).encode("UTF-8"))
        else:
            self.send_response(404)
            self.end_headers()


if __name__ == '__main__':
    httpd = HTTPServer(("", 4355), Plugin)
    httpd.serve_forever()
```

Somethings to note here:

* You only need to implement the calls you need. Return 404 and it won't be called again.
* The path is the RPC method name.
* The request body contains the parameters.
* The response body contains the result.

To be able to invoke a plugin, the plugin must be accessible from the controller via HTTP. The best way to do this is to
start the plugins as a sidecar container in the workflow controller, i.e. as a sidecar. Add the following to your
workflows controller's spec:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: workflow-controller
spec:
  template:
    spec:
      containers:
        - name: hello-controller-plugin
          args:
            - |
              import json
              from http.server import BaseHTTPRequestHandler, HTTPServer


              class Plugin(BaseHTTPRequestHandler):

                def do_POST(self):
                  if self.path == "/WorkflowLifecycleHook.WorkflowPreOperate":
                    print("hello")
                    self.send_response(200)
                    self.end_headers()
                    self.wfile.write(json.dumps({}).encode("UTF-8"))
                  else:
                    self.send_response(404)
                    self.end_headers()


              if __name__ == '__main__':
                httpd = HTTPServer(('', 4355), Plugin)
                httpd.serve_forever()

          command:
            - python
            - -c
          image: python:alpine3.6
```

The controller needs to be able discover its plugins. To do this create a config map:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: hello-controller-plugin
  labels:
    workflows.argoproj.io/configmap-type: ControllerPlugin
data:
  address: http://localhost:4355
```

Restart your controller and check the logs:

```
level=info msg="plugins" plugins=true
level=info msg="loading plugin" name=hello-controller-plugin
```

### Next Steps

* Take a look at
  an [advanced example](https://github.com/argoproj/argo-workflows/tree/dev-plugins/examples/plugins/controller/hello)
  that shows how to write a plugin template types.
* Read the [plugin reference](https://github.com/argoproj/argo-workflows/tree/dev-plugins/pkg/plugins/controller) to see
  what other operation there are.

### Considerations

### Failure Modes

An error in a plugin is typically contained as follows:

* Transient errors are ignored, and reconciliation aborted.
* For node lifecycle hooks, the node will error. The workflow therefore may fail.
* Other errors will result in an errored workflow.

#### Performance Is Important

Consider a workflows with 100k nodes, and then consider you have 5 plugins:

We'll make num(nodes) x num(plugins) calls.

So we have 500k network calls per loop. 
