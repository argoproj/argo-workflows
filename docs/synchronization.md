# Sychronization

![beta](assets/beta.svg)

> v2.10 and after

Feature Request: [2550](https://github.com/argoproj/argo/issues/2550)

## Introduction
Synchronization feature enables to control the parallel execution of the class of workflows or templates that needs to be 
rate limited with in namespace. User can have multiple rate limit configuration in `configmap` and that can be referred 
in workflow or template in workflow.   

E.g:
```yaml
apiVersion: v1
 kind: ConfigMap
metadata:
 name: my-config
data:
  workflow: "1" # Only one workflow can run at given time in particular namespace
  template: "2" # Two instance of template can run at a given time in particular namespace
```

### Worfklow level Synchronization

E.g:
```yaml
#
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

### Template level Synchronization

E.g:
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

###Existing Parallelism support:
WorkflowController already has a parallelism configuration in the controller.However, this setting applies to all workflows 
in the system, and is not granular to a class of workflows, or step. There is also a parallelism setting at a workflow and template level, but this only restricts total concurrent executions 
of steps from within the same workflow. The existing Parallelism support will be superseded with this feature. 

