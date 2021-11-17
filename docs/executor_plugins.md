# Executor Plugins

## Types

### Template Executor

Run custom ("plugin") templates, e.g for non-pod tasks such as Tekton builds or Spark jobs.

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

    def do_POST(self):
        if self.path == "/api/v1/template.execute":
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

The controller needs to be able discover its plugins. To do this create a config map in the user namespace:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: hello-executor-plugin
  labels:
    workflows.argoproj.io/configmap-type: ExecutorPlugin
data:
  address: http://localhost:4355
  image: python:alpine3.6
  command: |
    - python
    - -c
  args: |
    - |
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
                  if 'howdy' in args['template']['plugin']:
                      self.reply({'node': {'phase': 'Succeeded', 'message': 'Hello template!'}})
                  else:
                      self.reply({})
              else:
                  self.unsupported()


      if __name__ == '__main__':
          httpd = HTTPServer(('', 4355), Plugin)
          httpd.serve_forever()


```

### What Next?

- [Swagger](executor_swagger.md)
