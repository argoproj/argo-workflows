Description: Enable onExit handlers to access DAG task outputs via workflow.outputs.parameters
Author: [yeonsookim](https://github.com/yeonsookim)
Component: General
Issues: 14767

<!--
This feature enables onExit handlers to access DAG task outputs using `{{workflow.outputs.parameters.*}}` references.

### When to use this feature

* When you need to perform cleanup operations based on DAG task results
* When you want to send notifications that include task output values
* When you need conditional logic in onExit handlers based on task outcomes
* When you want to perform post-processing on task results

### Code examples

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: data-processing-with-cleanup
spec:
  entrypoint: main
  onExit: cleanup-handler
  templates:
  - name: main
    dag:
      tasks:
      - name: process-data
        template: data-processor
  - name: data-processor
    container:
      image: python:3.9
      command: [python, -c]
      args: ["print('Processing completed') > /tmp/result.txt"]
    outputs:
      parameters:
      - name: processing-status
        globalName: data-status
        valueFrom:
          path: /tmp/result.txt
  - name: cleanup-handler
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo 'Cleanup completed. Status: {{workflow.outputs.parameters.data-status}}'"]
```

### Technical details

* Runtime logic: Global parameters are updated before onExit handler execution
* Validation logic: `workflow.outputs.parameters.*` references are allowed in validation
* Backward compatibility: Existing workflows continue to work unchanged
* Coverage: All validation paths support the new pattern
-->
