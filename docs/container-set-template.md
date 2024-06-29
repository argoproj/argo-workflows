# Container Set Template

> v3.1 and after

A container set template is similar to a normal container or script template, but allows you to specify multiple
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

There are a couple of caveats:

1. You must use the [Emissary Executor](workflow-executors.md#emissary-emissary).
2. Or all containers must run in parallel - i.e. it is a graph with no dependencies.
3. You cannot use [enhanced depends logic](enhanced-depends-logic.md).
4. It will use the sum total of all resource requests, maybe costing more than the same DAG template. This will be a problem if your requests already cost a lot. See below.

The containers can be arranged as a graph by specifying dependencies. This is suitable for running 10s rather than 100s
of containers.

## Inputs and Outputs

As with the container and script templates, inputs and outputs can only be loaded and saved from a container
named `main`.

All container set templates that have artifacts must/should have a container named `main`.

If you want to use base-layer artifacts, `main` must be last to finish, so it must be the root node in the graph.

That may not be practical.

Instead, have a workspace volume and make sure all artifacts paths are on that volume.

## ⚠️ Resource Requests

A container set actually starts all containers, and the Emissary only starts the main container process when the containers it depends on have completed. This mean that even though the container is doing no useful work, it is still consuming resources and you're still getting billed for them.

If your requests are small, this won't be a problem.

If your requests are large, set the resource requests so the sum total is the most you'll need at once.

Example A: a simple sequence e.g. `a -> b -> c`

* `a` needs 1Gi memory
* `b` needs 2Gi memory
* `c` needs 1Gi memory

Then you know you need only a maximum of 2Gi. You could set as follows:

* `a` requests 512Mi memory
* `b` requests 1Gi memory
* `c` requests 512Mi memory

The total is 2Gi, which is enough for `b`. We're all good.

Example B: Diamond DAG e.g. a diamond `a -> b -> d and  a -> c -> d`, i.e. `b` and `c` run at the same time.

* `a` needs 1000 cpu
* `b` needs 2000 cpu
* `c` needs 1000 cpu
* `d` needs 1000 cpu

I know that `b` and `c` will run at the same time. So I need to make sure the total is 3000.

* `a` requests 500 cpu
* `b` requests 1000 cpu
* `c` requests 1000 cpu
* `d` requests 500 cpu

The total is 3000, which is enough for `b + c`. We're all good.

Example B: Lopsided requests, e.g. `a -> b` where `a` is cheap and `b` is expensive

* `a` needs 100 cpu, 1Mi memory, runs for 10h
* `b` needs 8Ki GPU, 100 Gi memory, 200 Ki GPU, runs for 5m

Can you see the problem here? `a` only has small requests, but the container set will use the  total of all requests. So it's as if you're using all that GPU for 10h. This will be expensive.

Solution: do not use container set when you have lopsided requests.

## Inner `retryStrategy` usage

> v3.3 and after

You can set an inner `retryStrategy` to apply to all containers of a container set, including the `duration` between each retry and the total number of `retries`.

See an example below:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: containerset-with-retrystrategy
  annotations:
    workflows.argoproj.io/description: |
      This workflow creates a container set with a retryStrategy.
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

!!! Note "Template-level `retryStrategy` vs Container Set `retryStrategy`"
    `containerSet.retryStrategy` works differently from [template-level retries](retries.md):

    1. Your `command` will be re-ran by the Executor inside the same container if it fails.

        - As no new containers are created, the nodes in the UI remain the same, and the retried logs are appended to original container's logs. For example, your container logs may look like:
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

    1. If a container's `command` cannot be located, it will not be retried.

        - As it will fail each time, the retry logic is short-circuited.

<!-- markdownlint-enable MD046 -->
