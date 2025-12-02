# Suspending

Workflows can be suspended by

```bash
argo suspend WORKFLOW
```

Or by specifying a `suspend` step on the workflow:

/// tab | YAML

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: suspend-template-
spec:
  entrypoint: suspend
  templates:
  - name: suspend
    steps:
    - - name: build
        template: hello-world
    - - name: approve
        template: approve
    - - name: delay
        template: delay
    - - name: release
        template: hello-world

  - name: approve
    suspend: {}

  - name: delay
    suspend:
      duration: "20"    # Must be a string. Default unit is seconds. Could also be a Duration, e.g.: "2m", "6h"

  - name: hello-world
    container:
      image: busybox
      command: [echo]
      args: ["hello world"]
```

///

/// tab | Python

```python
from hera.workflows import Container, Steps, Suspend, Workflow

with Workflow(
    generate_name="suspend-template-",
    entrypoint="suspend",
) as w:
    approve = Suspend(name="approve")
    delay = Suspend(name="delay", duration="20")
    hello_world = Container(
        name="hello-world",
        image="busybox",
        command=["echo"],
        args=["hello world"],
    )
    with Steps(name="suspend") as steps:
        hello_world(name="build")
        approve()
        delay()
        hello_world(name="release")
```

///
Once suspended, a Workflow will not schedule any new steps until it is resumed. It can be resumed manually by

```bash
argo resume WORKFLOW
```

Or automatically with a `duration` limit as the example above.
