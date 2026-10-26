# Executor Plugins

## Configuration

Executor Plugins are disabled by default.
They can be enabled either globally via the Workflow Controller ConfigMap or per workflow.
Use the corresponding flags to enable them:

- `ARGO_EXECUTOR_PLUGINS=true` enables the use of Executor Plugins defined globally in the controller’s ConfigMap.
- `ARGO_WORKFLOW_LEVEL_EXECUTOR_PLUGINS=true` allows using Executor Plugin settings specified directly in the workflow spec. These settings take precedence over the global ones.

These two options are independent of each other: you can enable only the global plugins, only the workflow-level plugins, or both at the same time.

See [`workflow-level-executor-plugin.yaml`](https://github.com/argoproj/argo-workflows/blob/main/examples/workflow-level-executor-plugin.yaml) for a workflow-level executor plugin example.

For example, to enable both modes in the Workflow Controller deployment:

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
            - name: ARGO_WORKFLOW_LEVEL_EXECUTOR_PLUGINS
              value: "true"
```

When using the [Helm chart](https://github.com/argoproj/argo-helm/tree/master/charts/argo-workflows), add this to your `values.yaml`:

```yaml
controller:
  extraEnv:
    - name: ARGO_EXECUTOR_PLUGINS
      value: "true"
    - name: ARGO_WORKFLOW_LEVEL_EXECUTOR_PLUGINS
      value: "true"
```

### Example: A Simple Python Plugin

Let's make a Python Executor Plugin that prints "hello" each time the Workflow is operated on.

We need the following:

1. Executor Plugins enabled on the Workflow Controller (see [configuration](#configuration)).
1. [Additional RBAC permissions to run and manage the Agent Pod](http-template.md#argo-agent-rbac).
1. A HTTP server that will be run as a sidecar to the main container and will respond to RPC HTTP requests from the executor with [this API contract](executor_swagger.md).
1. A `plugin.yaml` configuration file, that is turned into a ConfigMap so the controller can discover the plugin.

A template Executor Plugin services HTTP POST requests on `/api/v1/template.execute`:

```bash
curl http://localhost:4355/api/v1/template.execute -d \
'{
  "workflow": {
    "metadata": {
      "name": "my-wf"
    }
  },
  "template": {
    "name": "my-tmpl",
    "inputs": {},
    "outputs": {},
    "plugin": {
      "hello": {}
    }
  }
}'
# ...
HTTP/1.1 200 OK
{
  "node": {
    "phase": "Succeeded",
    "message": "Hello template!"
  }
}
```

**Tip:** The port number can be anything, but must not conflict with other Executor Plugins.
Don't use common ports such as 80, 443, 8080, 8081, 8443.
If you plan to publish your plugin, choose a random port number under 10000 and create a PR to add your plugin.
If not, use a port number greater than 10000.

We'll need to create a script that starts a HTTP server.
Save this as `server.py`:

```python
import json
from http.server import BaseHTTPRequestHandler, HTTPServer

with open("/var/run/argo/token") as f:
    token = f.read().strip()


class Plugin(BaseHTTPRequestHandler):

    def args(self):
        return json.loads(self.rfile.read(int(self.headers.get('Content-Length'))))

    def reply(self, reply):
        self.send_response(200)
        self.end_headers()
        self.wfile.write(json.dumps(reply).encode("UTF-8"))

    def forbidden(self):
        self.send_response(403)
        self.end_headers()

    def unsupported(self):
        self.send_response(404)
        self.end_headers()

    def do_POST(self):
        if self.headers.get("Authorization") != "Bearer " + token:
            self.forbidden()
        elif self.path == '/api/v1/template.execute':
            args = self.args()
            if 'hello' in args['template'].get('plugin', {}):
                self.reply(
                    {'node': {'phase': 'Succeeded', 'message': 'Hello template!',
                              'outputs': {'parameters': [{'name': 'foo', 'value': 'bar'}]}}})
            else:
                self.reply({})
        else:
            self.unsupported()


if __name__ == '__main__':
    httpd = HTTPServer(('', 4355), Plugin)
    httpd.serve_forever()
```

**Tip**: Executor Plugins can be written in any language you can run as a container.
Python is convenient because you can embed
the script in the container.

Some things to note here:

- You only need to implement the calls you need. Return 404 and it won't be called again.
- The path is the RPC method name.
- You should check that the `Authorization` header contains the same value as `/var/run/argo/token`. Return 403 if not.
- The request body contains the template's input parameters.
- The response body may contain the node's result, including the phase (e.g. "Succeeded" or "Failed") and a message.
- If the response is `{}`, then the Executor Plugin is saying it cannot execute the Plugin template, e.g. it is a Slack Plugin, but the template is a Tekton job.
- If the status code is 404, then the Executor Plugin will not be called again.
- If you save the file as `server.*`, it will be copied to the sidecar container's `args` field.
  This is useful for building self-contained Executor Plugins in scripting languages like Python or Node.JS.

Next, create a manifest named `plugin.yaml`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ExecutorPlugin
metadata:
  name: hello
spec:
  sidecar:
    container:
      command:
        - python
        - -u # disables output buffering
        - -c
      image: python:alpine3.23
      name: hello-executor-plugin
      ports:
        - containerPort: 4355
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534 # nobody
      resources:
        requests:
          memory: "64Mi"
          cpu: "250m"
        limits:
          memory: "128Mi"
          cpu: "500m"
```

Build and install as follows:

```bash
argo executor-plugin build .
kubectl -n argo apply -f hello-executor-plugin-configmap.yaml
```

Check your controller logs:

```text
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
        hello: {}
```

You'll see the Workflow complete successfully.

### Discovery

When a workflow is run, Executor Plugins are loaded from:

- The Workflow's namespace.
- The Argo installation namespace (typically `argo`).

If two Executor Plugins have the same name, only the one in the Workflow's namespace is loaded.

### Secrets

If you interact with a third-party system, you'll need access to secrets.
Don't put them in `plugin.yaml`.
Use a secret:

```yaml
spec:
  sidecar:
    container:
      env:
        - name: URL
          valueFrom:
            secretKeyRef:
              name: slack-executor-plugin
              key: URL
```

Refer to the [Kubernetes Secret documentation](https://kubernetes.io/docs/concepts/configuration/secret/) for secret best practices and security considerations.

### Resources, Security Context

We made these mandatory, so no one can create an Executor Plugin that uses an unreasonable amount of memory, or run as root unless
they deliberately do so:

```yaml
spec:
  sidecar:
    container:
      resources:
        requests:
          cpu: 100m
          memory: 32Mi
        limits:
          cpu: 200m
          memory: 64Mi
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
```

### Failure

An Executor Plugin may fail as follows:

- Connection/socket error - considered transient.
- Timeout - considered transient.
- 404 error - method is not supported by the Executor Plugin, as a result the method will not be called again (in the same workflow).
- 503 error - considered transient.
- Other 4xx/5xx errors - considered fatal.

Transient errors are retried, all other errors are considered fatal.

Fatal errors will result in failed steps.

### Re-Queue

It might be the case that the Executor Plugin can't finish straight away.
For example, it starts a long running task.
When that happens, you return "Pending" or "Running" and a re-queue time:

```json
{
  "node": {
    "phase": "Running",
    "message": "Long-running task started"
  },
  "requeue": "2m"
}
```

In this example, the task will be re-queued and `template.execute` will be called again in 2 minutes.

### Artifacts

A Plugin template can have input and output artifacts, as a container template can:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: plugin-artifacts-
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: produce
            template: produce
        - - name: consume
            template: consume
            arguments:
              artifacts:
                - name: report
                  from: "{{steps.produce.outputs.artifacts.report}}"
    - name: produce
      plugin:
        hello: {}
      outputs:
        artifacts:
          - name: report
            path: /tmp/report.txt
    - name: consume
      plugin:
        hello: {}
      inputs:
        artifacts:
          - name: report
            path: /tmp/report.txt
```

The Agent and its Executor Plugins share a workspace volume, mounted at `/argo/workspace` in each of their containers.
A node with artifacts gets a directory of its own in the workspace, which its artifact paths are relative to.
Before the Agent offers the node to the Executor Plugins, it loads the input artifacts into the directory.
In the template it sends, each artifact's `path` is the absolute path of that artifact in the directory, for example `/argo/workspace/plugin-artifacts-1234567890/tmp/report.txt`.
The Executor Plugin reads its input artifacts from, and writes its output artifacts to, those paths:

```python
template = args['template']
for artifact in template.get('inputs', {}).get('artifacts', []):
    with open(artifact['path']) as f:
        data = f.read()
for artifact in template.get('outputs', {}).get('artifacts', []):
    with open(artifact['path'], 'w') as f:
        f.write('Hello')
```

See [`executor-plugin-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/main/examples/executor-plugin-artifacts.yaml) for a complete example.

When the Executor Plugin replies with a `Succeeded`, `Failed` or `Error` phase, the Agent uploads the output artifacts to the [artifact repository](configure-artifact-repository.md), as the `wait` container does for a container template.
Archive strategies, keys, optional artifacts and [artifact garbage collection](walk-through/artifacts.md#artifact-garbage-collection) work as they do for container templates.
A required output artifact that is missing turns a `Succeeded` node into an `Error` one.
`{{pod.name}}` in the artifact repository's key format is the name the node's Pod would have had.

An Executor Plugin can also store an output artifact itself, and reply with its location:

```json
{
  "node": {
    "phase": "Succeeded",
    "outputs": {
      "artifacts": [{ "name": "report", "s3": { "key": "path/to/report.txt" } }]
    }
  }
}
```

The Agent leaves an output artifact the Executor Plugin replied with a location or key for as it is.

Some things to note here:

- The Agent reads the secrets of the workflow's artifact repository from its Pod, where they are mounted.
  It reads any other secret, such as one an artifact with a location of its own refers to, from the Kubernetes API, which needs a [Role](http-template.md#argo-agent-rbac) granting `get` on it.
- [Artifact plugins](artifact-plugin.md) cannot store a Plugin template's artifacts.
- The Executor Plugins share the workspace, so each of them can read and change every node's directory, not only those of the nodes it runs.
  They cannot make the Agent read or write anything outside a node's directory: the Agent follows no symlink out of it, and downloads, archives and uploads artifacts in a directory of its own, which it copies them to and from.
- The Agent loads and saves a node's artifacts in one of its [`ARGO_AGENT_TASK_WORKERS`](environment-variables.md) workers, which a large artifact takes up until it is done.
  It does so within its resource limits, which [`ARGO_AGENT_CPU_LIMIT` and `ARGO_AGENT_MEMORY_LIMIT`](environment-variables.md) set.
- The Agent's own directory is in its `/tmp`, an `emptyDir` volume, which uses the Kubernetes Node's ephemeral storage.
  While the Agent saves an output artifact, the artifact can take up to twice its size there.
- The workspace is an `emptyDir` volume, which lasts as long as the Agent Pod.
  The Agent removes a node's directory when the node is done, except for what an Executor Plugin running as another user made in it.
  If the Agent's container restarts, it carries on with the nodes it was running, without loading their input artifacts again.
- An Executor Plugin's container cannot mount a volume at `/argo/workspace`.

## Debugging

You can find the Executor Plugin's log in the Agent pod's sidecar, e.g.:

```bash
kubectl -n argo logs ${agentPodName} -c hello-executor-plugin
```

## Listing Executor Plugins

Because Executor Plugins are just ConfigMaps, you can list them using `kubectl`:

```bash
kubectl get cm -l workflows.argoproj.io/configmap-type=ExecutorPlugin
```

## Examples and Community Contributed Executor Plugins

[Plugin directory](plugin-directory.md)

## Publishing Your Executor Plugin

If you want to publish and share your Executor Plugin (we hope you do!), then submit a Pull Request to add it to the above
directory.
