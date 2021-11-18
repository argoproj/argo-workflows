# Executor Plugins

## Types

### Template Executor

There is only one type of executor plugin, one that runs custom ("plugin") templates, e.g for non-pod tasks such as
Tekton builds or Spark jobs.

## A Simple Python Plugin

Lets make a Python plugin that prints "hello" each time the workflow is operated on.

We need the following:

1. A HTTP server that will be run as a sidecar to the main container responds to RPC HTTP requests from the executor.
2. Configuration so the controller can discover the plugin.

We'll need to create a script that starts a HTTP server:

```python
import json
from http.server import BaseHTTPRequestHandler, HTTPServer


class Plugin(BaseHTTPRequestHandler):

    def args(self):
        return json.loads(self.rfile.read(int(self.headers.get('Content-Length'))))

    def reply(self, reply):
        self.send_response(200)
        self.end_headers()
        self.wfile.write(json.dumps(reply).encode("UTF-8"))

    def unsupported(self):
        self.send_response(404)
        self.end_headers()

    def do_POST(self):
        if self.path == '/api/v1/template.execute':
            args = self.args()
            if 'howdy' in args['template'].get('plugin', {}):
                self.reply({'node': {'phase': 'Succeeded', 'message': 'Hello template!'}})
            else:
                self.reply({})
        else:
            self.unsupported()


if __name__ == '__main__':
    httpd = HTTPServer(('', 4355), Plugin)
    httpd.serve_forever()
```

Somethings to note here:

* You only need to implement the calls you need. Return 404 and it won't be called again.
* The path is the RPC method name.
* The request body contains the parameters.
* The response body contains the result.

Next, create a manifest named `plugin.yaml`:

```yaml
kind: ExecutorPlugin
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
    name: hello-executor-plugin
```

Build and install as follows:

```shell
argo plugin build .
kubectl -n argo apply -f hello-executor-configmap.yaml
```

Check your controller logs:

```
level=info msg="Executor plugin added" name=hello-controller-plugin

```

Run this workflow.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-
spec:
  entrypoint: main
  templates:
    - name: main
      plugin:
        howdy: { }
```

You'll see the workflow complete successfully.

### Debugging

You can find the plugin's log in the agent pod's sidecar, e.g.:

```
kubectl -n argo logs ${agentPodName} -c hello-executor-plugin
```

### Learn More

- Read the [API reference](executor_swagger.md).
