# Plugins

Plugins allow you to extend Argo Workflows.

## Controller Plugins

### Types

#### Workflow Lifecycle Hook

You can can view or modify a workflow before it is operated on, or saved.

Use cases:

* Validate it, e.g. to enforce image allow list.
* Record meta-data and lineage to external system.
* Real-time progress reporting.
* Emit notifications.

#### Node Lifecycle Hook

You can can view or modify a node before or after it is executed.

Use cases:

* Short-circuit executing, marking steps as complete based on external information.
* Run custom templates (if they can run in the operator's namespace).
* Run non-pod tasks, e.g Tekton or Spark jobs.
* Offload caching decision to an external system.
* Block workflow execution until an external system has finished logging some metadata for one task.
* Emit notifications.

#### Pod Lifecycle Hook

* Add labels or annotations to a pod.
* Add a sidecar to every pod.
* Prevent pods being created.

#### Parameters Substitution Plugin

* Allow extra placeholders using data from an external system.

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
        - name: rpc-1234
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
                httpd = HTTPServer(('', 1234), Plugin)
                httpd.serve_forever()

          command:
            - python
            - -c
          image: python:alpine3.6
```

You also need to enable the plugin by creating this configuration:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: rpc-1234-controller-plugin
  labels:
    workflows.argoproj.io/configmap-type: ControllerPlugin
data:
  path: rpc.so
  address: http://localhost:1234
```

Verify the controller started successfully, and logged:

```
level=info msg="loading plugin" path=/plugins/rpc.so
```

You can enable more of this plugin, by changing "1234" to "1235" etc.

### Authoring A Golang Plugin

Before you start, please
research [is anyone using Go plugins](https://www.google.com/search?client=safari&rls=en&q=is+anyone+using+go+plugins&ie=UTF-8&oe=UTF-8)
, downsides:

* Only Linux and MacOS, not Windows.
* You must re-build them for every new version of Argo Workflows.

Look at the example workflow/controller/plugins/hello/plugin.go.

#### Installing Shared Libraries

Shared libraries must be in the `/plugins` directory. The easiest option to get them into that directory is to add
an init container that either has the plugin as part of the image, or gets the plugin from elsewhere.

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

### Performance Is Important

Consider a workflows with 100k nodes, and then consider you have 5 plugins:

We'll make num(nodes) x num(plugins) calls.

So we have 500k network calls per loop. 

## Agent Plugins

An agent plugin allows the agent to do work for a workflow.

Use cases

* Run custom templates (if they can run in the user's namespace).