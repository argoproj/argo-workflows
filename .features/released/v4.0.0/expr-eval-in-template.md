Description: Evaluate expressions in workflow template parameters.
Author: [Erwan Daniel](https://github.com/k3rnL)
Component: UI
Issues: 14726

Expression defaults make `WorkflowTemplate` and `ClusterWorkflowTemplate` submit parameters dynamic.
A default wrapped in `{{= ... }}` is evaluated when the API server handles `GetWorkflowTemplate` or `GetClusterWorkflowTemplate`, rather than when the workflow runs.
The result is returned in the parameter's `value`, while `default` keeps the original expression and the stored template is not modified.
Workflow-level parameter defaults are evaluated first.
Template input defaults are evaluated next and can reference the evaluated workflow parameters through `workflow.parameters`.
An expression that is allowed to remain unresolved leaves `value` unset and keeps the raw expression in `default`; invalid expression syntax makes the Get request return an error.
For example, you can prefill submit parameters with today's date, making it easier for operators to start a job without typos.

```yaml
metadata:
  name: expression-defaults
  namespace: argo
spec:
  entrypoint: main
  arguments:
    parameters:
      - name: run-date
        default: "{{=now().Format('2006-01-02')}}"
  templates:
    - name: main
      inputs:
        parameters:
          - name: message
            default: "{{=sprig.printf('run for %s', workflow.parameters['run-date'])}}"
      container:
        image: argoproj/argosay:v2
        args:
          - echo
          - "{{inputs.parameters.message}}"
```
