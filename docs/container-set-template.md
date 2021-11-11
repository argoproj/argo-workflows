# Container Set Template

![alpha](assets/alpha.svg)

> v3.1 and after

A container set templates is similar to a normal container or script template, but allows you to specify multiple
containers to run within a single pod.

Because you have multiple containers within a pod, they will be scheduled on the same host. You can use cheap and fast
empty-dir volumes instead of persistent volume claims to share data between steps.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: container-set-template-
spec:
  entrypoint: main
  templates:
    - name: main
      volumes:
        - name: workspace
          emptyDir: { }
      containerSet:
        volumeMounts:
          - mountPath: /workspace
            name: workspace
        containers:
          - name: a
            image: argoproj/argosay:v2
          - name: b
            image: argoproj/argosay:v2
          - name: main
            image: argoproj/argosay:v2
            dependencies:
              - a
              - b
      outputs:
        parameters:
          - name: message
            valueFrom:
              path: /workpsace/message
```

There are a couple of caveats:

1. You must use the [Emissary Executor](workflow-executors.md#emissary-emissary).
2. Or all containers must run in parallel - i.e. it is a graph with no dependencies.

The containers can be arranged as a graph by specifying dependencies. This is suitable for running 10s rather than 100s
of containers.

## Inputs and Outputs

As with the container and script templates, inputs and outputs can only be loaded and saved from a container
named `main`.

All container set templates that have artifacts must/should have a container named `main`.

If you want to use base-layer artifacts, `main` must be last to finish, so it must be the root node in the graph.

That is may not be practical.

Instead, have a workspace volume and make sure all artifacts paths are on that volume.
