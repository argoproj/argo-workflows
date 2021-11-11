# HTTP Template
 
> v3.2 and after 

`HTTP Template` is a type of template which can execute the HTTP Requests.

 ### Agent Architecture
 V3.2 introduced `Agent` architecture to execute the multiple HTTPTemplates in single pod which improve a performance and resource utilization.
 `WorkflowTaskSet` CRD is introduced to exchange the data between Controller and Agent. 
 Agent pod named <workflowname-agent> and WorkflowTaskSet name as WorkflowName.

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
        - - name: good
            template: http
            arguments:
              parameters: [{name: url, value: "https://raw.githubusercontent.com/argoproj/argo-workflows/4e450e250168e6b4d51a126b784e90b11a0162bc/pkg/apis/workflow/v1alpha1/generated.swagger.json"}]
    - name: http
      inputs:
        parameters:
          - name: url
      http:
       # url: http://dummy.restapiexample.com/api/v1/employees
       url: "{{inputs.parameters.url}}"
      
```
