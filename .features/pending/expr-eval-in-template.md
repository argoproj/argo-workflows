Description: Evaluate expressions in workflow template parameters.
Author: [Erwan Daniel](https://github.com/k3rnL)
Component: UI
Issues: 14726

Expression defaults make `WorkflowTemplate` and `ClusterWorkflowTemplate` submit parameters dynamic.
A default wrapped in `{{= ... }}` is evaluated when the API server handles `GetWorkflowTemplate` or `GetClusterWorkflowTemplate`, rather than when the workflow runs.
The result is returned in the parameter's `value`, while `default` keeps the original expression and the stored template is not modified.
An explicit `value` always takes precedence over a `default`.
Workflow-level parameter defaults are evaluated first.
Template input defaults are evaluated next and can reference the evaluated workflow parameters through `workflow.parameters`.
An expression that is allowed to remain unresolved leaves `value` unset and keeps the raw expression in `default`; invalid expression syntax makes the Get request return an error.
For example, a run-date default such as `{{=now().Format('2006-01-02')}}` can prefill today's date, making it easier for operators to start a job without typos.
