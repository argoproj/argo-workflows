# Hello World

Run this container directly from your shell with a `docker` command:

```bash
$ docker run busybox echo "hello world"
hello world
```

Below, run the same container on a Kubernetes cluster with a Workflow.
The comments provide useful explanations.

/// tab | YAML

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow                  # new type of k8s spec
metadata:
  generateName: hello-world-    # name of the workflow spec
spec:
  entrypoint: hello-world       # invoke the hello-world template
  templates:
    - name: hello-world         # name of the template
      container:
        image: busybox
        command: [ echo ]
        args: [ "hello world" ]
        resources: # limit the resources
          limits:
            memory: 32Mi
            cpu: 100m
```

///

/// tab | Python

```py
from hera.workflows import Container, Resources, Workflow # (1)!

with Workflow(
    generate_name="hello-world-",
    entrypoint="hello-world",
) as w:
    Container(
        name="hello-world",
        image="busybox",
        command=["echo"],
        args=["hello world"],
        resources=Resources(memory_limit="32Mi", cpu_limit="100m"),
    )
```

1. Install the `hera` package to define your Workflows in Python. Learn more at [the Hera docs](https://hera.readthedocs.io/en/stable/).

///

Argo adds a new `kind` of Kubernetes resource called a `Workflow`.

The above spec contains a single `template` called `hello-world` which runs the `busybox` image and invokes `echo "hello world"`.
The `hello-world` template is the `entrypoint` for the spec.
The `entrypoint` specifies the first template to invoke when the workflow spec is executed.
Specifying the entrypoint is useful when there are multiple templates defined in the workflow spec.
