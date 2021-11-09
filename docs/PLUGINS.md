# Plugins

Plugins allow you to extend Argo Workflows.

## Controller Plugins

### Types

#### Workflow Lifecycle Hook

You can can view or modify a workflow before it is operated on, or saved.

Use cases:

* Validate it, e.g. to enforce so custom constraint.
* Record meta-data and lineage to external system.
* Real-time progress reporting.
* Emit notifications.

#### Node Lifecycle Hook

You can can view or modify a node before or after it is executed.

Use cases:

* Short-circuit executing, marking steps as complete based on external information.
* Run custom templates.
* Run non-pod tasks, e.g Tekton or Spark jobs.
* Offload caching decision to an external system.
* Block workflow execution until an external system has finished logging some metadata for one task.
* Emit notifications.

#### Parameters Substitution Plugin

* Allow extra placeholders using data from an external system.

### Configuration

Plugins are disabled by default. Start the controller with `ARGO_PLUGINS=true`.

### Bundled Plugins

We bundle `rpc.so`, plugin that makes delegates RPC calls by making requests over HTTP, allowing you to write no-Go
plugins, e.g. using Python.

To use this, you need to add a new sidecar to the workflow controller to accept those HTTP requests, here's a basic
example:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: workflow-controller
spec:
  template:
    spec:
      containers:
        - name: rpc-7584
          args:
            - |
              import json
              from http.server import BaseHTTPRequestHandler, HTTPServer


              class Plugin(BaseHTTPRequestHandler):

                def do_POST(self):
                  if self.path == "/WorkflowLifecycleHook.WorkflowPreOperate":
                    print("hello", self.path)
                    self.send_response(200)
                    self.end_headers()
                    self.wfile.write(json.dumps({}).encode("UTF-8"))
                  else:
                    self.send_response(404)
                    self.end_headers()


              if __name__ == '__main__':
                httpd = HTTPServer(('', 7584), Plugin)
                httpd.serve_forever()

          command:
            - python
            - -c
          image: python:alpine3.6
```

You also need create this configuration:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: rpc-7584-controller-plugin
  labels:
    workflows.argoproj.io/configmap-type: ControllerPlugin
data:
  path: rpc.so
  address: http://localhost:7584
```

Verify the controller started successfully, and logged:

```
level=info msg="loading plugin" path=/plugins/rpc.so
```

You can enable more of this plugin, by changing "7584" to "1234" etc.

You only need to implement the methods you need. If you return 404 error, then that method will not be called again.

### Authoring A Golang Plugin

Golang plugins have advantages over RPC sidecar plugins:

* Better at scale.
* Lower memory footprint.

But the have downsides too:

* Only Linux and MacOS, not Windows.
* Must be re-built for every new version of Argo Workflows.

[Is anyone using Go plugins?](https://www.google.com/search?client=safari&rls=en&q=is+anyone+using+go+plugins&ie=UTF-8&oe=UTF-8)

Look at the example workflow/controller/plugins/hello/plugin.go.

#### Installing Shared Libraries

Shared libraries must be in the `/plugins` directory. The easiest option to get them into that directory is to add an
init container that either has the plugin as part of the image, or gets the plugin from elsewhere.

The kustomize patch will get plugins.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: workflow-controller
spec:
  template:
    spec:
      initContainers:
        - name: plugins
          volumeMounts:
            - mountPath: /plugins
              name: plugins
          command:
            - sh
            - c
          args:
            - |
              curl http://.../plugin.tar.gz
              tar -xf plugins.tar.gz -C plugins
      containers:
        - name: workflow-controller
```

### Considerations

### Failure Modes

An error in a plugin is typically contained as follows:

* For node lifecycle hooks, the node will error. The workflow therefore may fail.
* Other errors will result in an errored workflow.
* Panics will result in an errored workflow.

Failures in a plugin should not take down the controller.

#### Performance Is Important

Consider a workflows with 100k nodes, and then consider you have 5 plugins:

We'll make num(nodes) x num(plugins) calls.

So we have 500k network calls per loop. 
