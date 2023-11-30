# HTTP Template

> v3.2 and after

`HTTP Template` is a type of template which can execute HTTP Requests.

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
        #  response.url: string, the request url
        #  response.method: string, the request method
        #  response.statusCode: int, the response status code
        #  response.body: string, the response body
        #  response.headers: map[string][]string, the response headers
        successCondition: "response.body contains \"google\"" # available since v3.3
        body: "test body" # Change request body
```

## Argo Agent

HTTP Templates use the Argo Agent, which executes the requests independently of the controller. The Agent and the Workflow
Controller communicate through the `WorkflowTaskSet` CRD, which is created for each running `Workflow` that requires the use
of the `Agent`.

In order to use the Argo Agent, you will need to ensure that you have added the appropriate [workflow RBAC](workflow-rbac.md) to add an agent role with to Argo Workflows. An example agent role can be found in [the quick-start manifests](https://github.com/argoproj/argo-workflows/tree/main/manifests/quick-start/base/agent-role.yaml).
