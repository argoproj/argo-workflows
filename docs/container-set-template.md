# ContainerSet Template

> v3.1 and after

A ContainerSet template is similar to a normal container or script template, but it allows you to run multiple containers within a single Pod.

Since multiple containers run within a single Pod, they schedule on the same host. You can use cheap and fast `emptyDir` volumes instead of PersistentVolumeClaims (PVCs) to share data between steps.

However, running all containers on the same host limits you to the host's resources.
Running all containers simultaneously may use more resources than running them sequentially.

Use ContainerSet templates strategically to avoid waiting for Pods to start and reduce the overhead of creating multiple init and wait containers.
This can be more efficient than using a DAG template.

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
            command: [sh, -c]
            args: ["echo 'a: hello world' >> /workspace/message"]
          - name: b
            image: argoproj/argosay:v2
            command: [sh, -c]
            args: ["echo 'b: hello world' >> /workspace/message"]
          - name: main
            image: argoproj/argosay:v2
            command: [sh, -c]
            args: ["echo 'main: hello world' >> /workspace/message"]
            dependencies:
              - a
              - b
      outputs:
        parameters:
          - name: message
            valueFrom:
              path: /workspace/message
```

There are a few caveats:

1. You cannot use [enhanced depends logic](enhanced-depends-logic.md).
1. The ContainerSet uses the sum total of all resource requests, which may cost more than using the same DAG template. This can be problematic if your [resource requests](#️-resource-requests) are already high.
1. ContainerSet templates can only run container templates.

You can arrange the containers as a graph by specifying dependencies.

ContainerSet templates are suitable for running tens rather than hundreds of containers.

## Inputs and Outputs

As with container and script templates, you can only load and save inputs and outputs from a container named `main`.

Include a container named `main` in all ContainerSet templates that have artifacts.

If you want to use base-layer artifacts, ensure `main` is the last to finish, making it the root node in the graph.
This may not always be practical.

Instead, use a workspace volume and ensure all artifact paths are on that volume.

## ⚠️ Resource Requests

A ContainerSet starts all containers, and the [workflow executor](workflow-executors.md#emissary-emissary) only starts the main container process when the containers it depends on have completed.

This means that even though the container is doing no useful work, it still consumes resources and you are still billed for them.

If your requests are small, this won't be a problem.

If your requests are large, set the resource requests so the sum total is the most you'll need at once.

### Example A: Simple Sequence (a -> b -> c)

* `a` needs 1Gi memory
* `b` needs 2Gi memory
* `c` needs 1Gi memory

You need a maximum of 2Gi. You could set the requests as follows:

* `a` requests 512Mi memory
* `b` requests 1Gi memory
* `c` requests 512Mi memory

The total is 2Gi, which is enough for `b`.

### Example B: Diamond DAG (a -> b -> d and a -> c -> d)

* `a` needs 1000 CPU
* `b` needs 2000 CPU
* `c` needs 1000 CPU
* `d` needs 1000 CPU

You know that `b` and `c` will run at the same time. So you need to make sure the total is 3000 CPU.

* `a` requests 500 CPU
* `b` requests 1000 CPU
* `c` requests 1000 CPU
* `d` requests 500 CPU

The total is 3000 CPU, which is enough for `b + c`.

### Example C: Lopsided Requests (a -> b)

* `a` needs 100 CPU, 1Mi memory, runs for 10h
* `b` needs 8Ki GPU, 100Gi memory, 200Ki GPU, runs for 5m

In this case, `a` only has small requests, but the ContainerSet uses the total of all requests. So it's as if you're using all that GPU for 10h. This will be expensive.
This is a good example of when using a ContainerSet would not be efficient.

## Inner `retryStrategy` usage

> v3.3 and after

Set an inner `retryStrategy` to apply to all containers of a container set, including the `duration` between each retry and the total number of `retries`.

See the example below:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: containerset-with-retrystrategy
  annotations:
    workflows.argoproj.io/description: |
      This workflow creates a ContainerSet template with a retryStrategy.
spec:
  entrypoint: containerset-retrystrategy-example
  templates:
    - name: containerset-retrystrategy-example
      containerSet:
        retryStrategy:
          retries: "10" # if fails, retry at most ten times
          duration: 30s # retry for at most 30s
        containers:
          # this container completes successfully, so it won't be retried.
          - name: success
            image: python:alpine3.6
            command:
              - python
              - -c
            args:
              - |
                print("hi")
          # if fails, it will retry at most ten times.
          - name: fail-retry
            image: python:alpine3.6
            command: ["python", -c]
            # fail with a 66% probability
            args: ["import random; import sys; exit_code = random.choice([0, 1, 1]); sys.exit(exit_code)"]
```

<!-- markdownlint-disable MD046 -- allow indentation within the admonition -->

!!! Note "Template-level `retryStrategy` vs ContainerSet `retryStrategy`"
    `containerSet.retryStrategy` works differently from [template-level retries](retries.md):

    1. The Executor re-runs your `command` inside the same container if it fails.

        - Since no new containers are created, the nodes in the UI remain the same, and the retried logs are appended to the original container's logs. For example, your container logs may look like:
          ```text
          time="2024-03-29T06:40:25 UTC" level=info msg="capturing logs" argo=true
          intentional failure
          time="2024-03-29T06:40:25 UTC" level=debug msg="ignore signal child exited" argo=true
          time="2024-03-29T06:40:26 UTC" level=info msg="capturing logs" argo=true
          time="2024-03-29T06:40:26 UTC" level=debug msg="ignore signal urgent I/O condition" argo=true
          intentional failure
          time="2024-03-29T06:40:26 UTC" level=debug msg="ignore signal child exited" argo=true
          time="2024-03-29T06:40:26 UTC" level=debug msg="forwarding signal terminated" argo=true
          time="2024-03-29T06:40:27 UTC" level=info msg="sub-process exited" argo=true error="<nil>"
          time="2024-03-29T06:40:27 UTC" level=info msg="not saving outputs - not main container" argo=true
          Error: exit status 1
          ```

    2. If the Executor cannot locate a container's `command`, it will not retry.

        - Since it will fail each time, the retry logic is short-circuited.

<!-- markdownlint-enable MD046 -->
