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

Let's make a Python plugin that prints "hello" each time the workflow is operated on.

You need the following:

1. A HTTP server that responds to RPC HTTP requests from the controller.
2. Configuration so the controller can discover the plugin.

Create a script that starts a HTTP server:

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

Create a manifest name `plugin.yaml`:

```yaml
kind: ControllerPlugin
metadata:
  name: hello
spec:
  address: http://localhost:4355
  description: This is the "hello world" plugin
  container:
    command:
      - python
      - -c
    image: python:alpine3.6
    name: hello-controller-plugin
```

Build and install:

```shell
argo plugin build .
kubectl -n argo apply -f hello-controller-configmap.yaml
kubectl -n argo patch deployment workflow-controller --patch-file hello-controller-plugin-deployment-patch.yaml
```

Check your controller logs:

```
level=info msg="Controller plugin added" name=hello-controller-plugin
```

Finally, run any workflow. You'll see "hello" printed in the plugins sidecar's logs.

```
kubectl -n argo logs deployment/workflow-controller -c hello-controller-plugin
```

## Learn More

* Take a look at
  an [advanced example](https://github.com/argoproj/argo-workflows/tree/dev-plugins/examples/plugins/controller/hello)
  that shows how to write a plugin template types.
* Read the [controller plugin API reference](controller_swagger.md) to see what other operations there are.

