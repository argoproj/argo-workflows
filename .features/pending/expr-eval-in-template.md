Description: Evaluate expressions in workflow template parameters.
Author: [Erwan Daniel](https://github.com/k3rnL)
Component: UI
Issues: 14726

This feature is useful when you want to make your `WorkflowTemplates` more dynamic.
For example, you can prefill your submit parameters with the date of today.
Making it easier for operators to start a job and avoid mistakes.

ex.
```yaml
metadata:
  name: omniscient-whale
  namespace: argo
spec:
  templates:
    - name: whalesay-template
      inputs:
        parameters:
          - name: message
            default: "{{workflow.parameters.test}}"
      outputs: {}
      metadata: {}
      container:
        name: ""
        image: docker/whalesay
        command:
          - cowsay
        args:
          - "{{inputs.parameters.message}}"
        resources: {}
  entrypoint: whalesay-template
  arguments:
    parameters:
      - name: test
        default: "{{=now().Format('2006-01-02 15:04:05')}}"
```