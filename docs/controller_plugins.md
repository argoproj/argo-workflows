# Controller Plugins

## Types

### Workflow Lifecycle Hook

View or modify a workflow before or after in is operated on.

Use cases:

* Validate it, e.g. to enforce so custom constraint.
* Record meta-data and lineage to external system.
* Real-time progress reporting.
* Notify users that a workflow completed or errored out.

### Node Lifecycle Hook

View or modify a node before or after it is executed, including writing your own custom templates.

Use cases:

* Short-circuit execution, marking steps as complete based on external information.
* Run custom ("plugin") templates, e.g for non-pod tasks such as Tekton builds or Spark jobs.
* Offload caching decision to an external system.
* Notify users when a particular step completes.

### Parameters Substitution Plugin

* Allow extra placeholders using data from an external system.

## A Simple Python Plugin

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
        if self.path == "/api/v1/workflow.preOperate":
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
                  if self.path == "/api/v1/node.preOperate":
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

## Next Steps

* Take a look at
  an [advanced example](https://github.com/argoproj/argo-workflows/tree/dev-plugins/examples/plugins/controller/hello)
  that shows how to write a plugin template types.
* Read the [controller plugin API reference](controller_swagger.md) to see what other operations there are.

