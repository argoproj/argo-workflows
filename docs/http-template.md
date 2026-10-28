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

## WorkflowTaskSet Size Limit

The WorkflowTaskSet holds the full template of every HTTP and Plugin node that is pending or running in a Workflow, so a wide fan-out of these templates can exceed the Kubernetes object size limit (about 1.5 MiB with the default etcd configuration).
When the controller cannot write new tasks to the WorkflowTaskSet because the WorkflowTaskSet would be too large, it marks every one of those nodes `Error` with a message naming the size limit, including any that would have fit on their own.
Tasks already in the WorkflowTaskSet keep running.
[`parallelism`](synchronization.md#workflow-level-parallelism) does not count HTTP or Plugin nodes, so it does not limit this fan-out.
To stay below the limit, use a [template-level semaphore](synchronization.md#template-level-synchronization) to run fewer HTTP or Plugin nodes at once, or reduce the size of their templates.
