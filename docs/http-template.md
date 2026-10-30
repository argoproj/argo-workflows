# HTTP Template

`HTTP Template` is a type of template which can execute HTTP Requests.
The body of the response is automatically exported into the `result` output parameter.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: http-template-
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: get-google-homepage
            template: http
            arguments:
              parameters: [{name: url, value: "https://www.google.com"}]
    - name: http
      inputs:
        parameters:
          - name: url
      http:
        timeoutSeconds: 20 # Default 30
        url: "{{inputs.parameters.url}}"
        method: "GET" # Default GET
        headers:
          - name: "x-header-name"
            value: "test-value"
        # Template will succeed if evaluated to true, otherwise will fail
        # Available variables:
        #  request.body: string, the request body
        #  request.headers: map[string][]string, the request headers
        #  request.url: string, the request url
        #  request.method: string, the request method
        #  response.statusCode: int, the response status code
        #  response.body: string, the response body
        #  response.headers: map[string][]string, the response headers
        successCondition: "response.body contains \"google\""
        body: "test body" # Change request body
```

## Argo Agent RBAC

HTTP and Plugin Templates use the Argo Agent, which executes the requests independently of the controller.
The Agent and the Workflow Controller communicate through the `WorkflowTaskSet` CRD, which is created for each running `Workflow` that requires the use of the `Agent`.

In order to use the Argo Agent, you will need to ensure that you have added the appropriate [workflow RBAC](workflow-rbac.md) to add an agent role with to Argo Workflows.
An example agent role can be found in [the quick-start manifests](https://github.com/argoproj/argo-workflows/tree/main/manifests/quick-start/base/agent-role.yaml).

## Argo Agent Pod Failures

Each Workflow that uses the Agent runs it in one pod.
If the Agent's container exits with an error, the kubelet restarts it, and the restarted Agent runs again every HTTP or Plugin task that has no result yet.
If the pod itself fails, for example because the node evicts it, the controller deletes the pod and creates a replacement, which also runs every task that has no result yet.
A request or plugin call can therefore run more than once, so make it safe to repeat.

The controller replaces a failed pod no sooner than one minute after that pod was created, and records an `AgentPodFailed` Warning event on the Workflow, with the reason the pod failed, each time it does.
There is no limit on the number of replacements: as with a crashing container, a task that never gets a result fails by the Workflow's `activeDeadlineSeconds`, or by its template `timeout` unless its plugin has reported it `Running`.
Without either setting, the task can run until the Workflow stops for another reason.
When the Workflow is stopping, terminating, or past its `activeDeadlineSeconds`, the controller errors the nodes of the tasks that have no result instead.
