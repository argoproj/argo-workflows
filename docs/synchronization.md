# Synchronization

![GA](assets/ga.svg)

> v2.10 and after

## Introduction
Synchronization enables users to limit the parallel execution of certain workflows or 
templates within a workflow without having to restrict others.

Users can create multiple synchronization configurations in the `ConfigMap` that can be referred to 
from a workflow or template within a workflow.

For example:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
 name: my-config
data:
  workflow: "1"  # Only one workflow can run at given time in particular namespace
  template: "2"  # Two instance of template can run at a given time in particular namespace
```

### Workflow-level Synchronization
Workflow-level synchronization limits parallel execution of the workflow if workflow have same synchronization reference. 
In this example, Workflow refers `workflow` synchronization key which is configured as rate limit 1, 
so only one workflow instance will be executed at given time even multiple workflows created. 

example:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: synchronization-wf-level-
spec:
  entrypoint: whalesay
  synchronization:
    semaphore:
      configMapKeyRef:
        name: my-config
        key: workflow
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
```

### Template-level Synchronization
Template-level synchronization limits parallel execution of the template across workflows, if template have same synchronization reference. 
In this example, `acquire-lock` template has synchronization reference of `template` key which is configured as rate limit 2, 
so, two instance of templates will be executed at given time even multiple step/task with in workflow or different workflow refers same template. 

example:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: synchronization-tmpl-level-
spec:
  entrypoint: synchronization-tmpl-level-example
  templates:
  - name: synchronization-tmpl-level-example
    steps:
    - - name: synchronization-acquire-lock
        template: acquire-lock
        arguments:
          parameters:
          - name: seconds
            value: "{{item}}"
        withParam: '["1","2","3","4","5"]'

  - name: acquire-lock
    synchronization:
      semaphore:
        configMapKeyRef:
          name: my-config
          key: template
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["sleep 10; echo acquired lock"]
```
Examples:
1. [Workflow level](https://github.com/argoproj/argo/blob/master/examples/synchronization-wf-level.yaml)
2. [Step level](https://github.com/argoproj/argo/blob/master/examples/synchronization-tmpl-level.yaml)

### Other Parallelism support:
In addition to this synchronization, the workflow controller supports a parallelism setting that applies to all workflows 
in the system (it is not granular to a class of workflows, or tasks withing them). Furthermore, there is a parallelism setting 
at the workflow and template level, but this only restricts total concurrent executions of tasks within the same workflow.


