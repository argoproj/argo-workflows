# Synchronization

> v2.10 and after

## Introduction

Synchronization enables users to limit the parallel execution of certain workflows or
templates within a workflow without having to restrict others.

Users can create multiple synchronization configurations in the `ConfigMap` that can be referred to
from a workflow or template within a workflow. Alternatively, users can
configure a mutex to prevent concurrent execution of templates or
workflows using the same mutex.

For example:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
 name: my-config
data:
  workflow: "1"  # Only one workflow can run at given time in particular namespace
  template: "2"  # Two instances of template can run at a given time in particular namespace
```

### Workflow-level Synchronization

Workflow-level synchronization limits parallel execution of the workflow if workflows have the same synchronization reference.
In this example, Workflow refers to `workflow` synchronization key which is configured as limit 1,
so only one workflow instance will be executed at given time even multiple workflows created.

Using a semaphore configured by a `ConfigMap`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: synchronization-wf-level-
spec:
  entrypoint: hello-world
  synchronization:
    semaphore:
      configMapKeyRef:
        name: my-config
        key: workflow
  templates:
  - name: hello-world
    container:
      image: busybox
      command: [echo]
      args: ["hello world"]
```

Using a mutex:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: synchronization-wf-level-
spec:
  entrypoint: hello-world
  synchronization:
    mutex:
      name: workflow
  templates:
  - name: hello-world
    container:
      image: busybox
      command: [echo]
      args: ["hello world"]
```

### Template-level Synchronization

Template-level synchronization limits parallel execution of the template across workflows, if templates have the same synchronization reference.
In this example, `acquire-lock` template has synchronization reference of `template` key which is configured as limit 2,
so two instances of templates will be executed at a given time: even multiple steps/tasks within workflow or different workflows referring to the same template.

Using a semaphore configured by a `ConfigMap`:

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

Using a mutex:

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
      mutex:
        name: template
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["sleep 10; echo acquired lock"]
```

Examples:

1. [Workflow level semaphore](https://github.com/argoproj/argo-workflows/blob/main/examples/synchronization-wf-level.yaml)
1. [Workflow level mutex](https://github.com/argoproj/argo-workflows/blob/main/examples/synchronization-mutex-wf-level.yaml)
1. [Step level semaphore](https://github.com/argoproj/argo-workflows/blob/main/examples/synchronization-tmpl-level.yaml)
1. [Step level mutex](https://github.com/argoproj/argo-workflows/blob/main/examples/synchronization-mutex-tmpl-level.yaml)

### Other Parallelism support

In addition to this synchronization, the workflow controller supports a parallelism setting that applies to all workflows
in the system (it is not granular to a class of workflows, or tasks withing them). Furthermore, there is a parallelism setting
at the workflow and template level, but this only restricts total concurrent executions of tasks within the same workflow.
