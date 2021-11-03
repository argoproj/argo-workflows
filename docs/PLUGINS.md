# Plugins

Plugins allow you to extend Argo Workflows.

## Types

## > Workflow Lifecycle Hook

You can can view or modify a workflow before it is operated on, or saved.

Use cases:

* Validate it, e.g. to enforce image allow list.
* Record meta-data and lineage to external system.
* Real-time progress reporting.
* Emit notifications.

### Node Lifecycle Hook

You can can view or modify a node before or after it is executed.

Use cases:

* Short-circuit executing, marking steps as complete based on external information.
* Run custom templates.
* Run non-pod tasks, e.g Tekton or Spark jobs.
* Offload caching decision to an external system.
* Block workflow execution until an external system has finished logging some metadata for one task.
* Emit notifications.

### Pod Lifecycle Hook

* Add labels or annotations to a pod.
* Add a sidecar to every pod.
* Prevent pods being created.

### Parameters Substitution Plugin

* Allow extra placeholders using data from an external system.

## Writing A Plugin

Plugins are Go plugins (please
research [is anyone using Go plugins](https://www.google.com/search?client=safari&rls=en&q=is+anyone+using+go+plugins&ie=UTF-8&oe=UTF-8)):

* Only Linux and MacOS, not Windows.
* You must re-build them for every new version of Argo Workflows.

Look at the example workflow/controller/plugins/hello/plugin.go.

## Installing Plugins

Plugins must be in the `plugins` directory. The easiest option to get them into that directory is to add an init
container that either has the plugin as part of the image, or gets the plugin from elsewhere.

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

## Non-Golang Plugins

> Why not an non-Golang/RPC sidecar plugin?

You can build a RPC sidecar as a Go plugin, but not vice-versa.

I think we can anticipate that you will want a non-Golang (esp. Python)
plugins: [#7154](https://github.com/argoproj/argo-workflows/issues/7154)

## Performance Is Important

Consider a workflows with 100k nodes, and then consider you have 5 plugins:

We'll make num(nodes) x num(plugins) calls.

So we have 500k network calls per loop. 
