# Workflow Progress

> v2.12 and after

When you run a workflow, the controller will report on its progress.

We define progress as two numbers, `N/M` such that `0 <= N <= M and 0 <= M`.

* `N` is the number of completed tasks.
* `M` is the total number of tasks.

E.g. `0/0`, `0/1` or `50/100`.

Unlike [estimated duration](estimated-duration.md), progress is deterministic. I.e. it will be the same for each workflow, regardless of any problems.

Progress for each node is calculated as follows:

1. For a pod node either `1/1` if completed or `0/1` otherwise.
2. For non-leaf nodes, the sum of its children.

For a whole workflow's, progress is the sum of all its leaf nodes.

!!! Warning
    `M` will increase during workflow run each time a node is added to the graph.

## Self reporting progress

> v3.3 and after

Pods in a workflow can report their own progress during their runtime. This self reported progress overrides the
auto-generated progress.

Reporting progress works as follows:

* create and write the progress to a file indicated by the env variable `ARGO_PROGRESS_FILE`
* format of the progress must be `N/M`

The executor will read this file every 3s and if there was an update,
patch the pod annotations with `workflows.argoproj.io/progress: N/M`.
The controller picks this up and writes the progress to the appropriate Status properties.

Initially the progress of a workflows' pod is always `0/1`. If you want to influence this, make sure to set an initial
progress annotation on the pod:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: progress-
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: progress
            template: progress
    - name: progress
      metadata:
        annotations:
          workflows.argoproj.io/progress: 0/100
      container:
        image: alpine:3.14
        command: [ "/bin/sh", "-c" ]
        args:
          - |
            for i in `seq 1 10`; do sleep 10; echo "$(($i*10))"'/100' > $ARGO_PROGRESS_FILE; done
```
