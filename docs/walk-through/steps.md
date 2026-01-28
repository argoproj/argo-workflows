# Steps

You can create multi-step workflows and nested workflows, as well as define more than one template in a workflow.
See the comments in the example below:

/// tab | YAML

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: steps-
spec:
  entrypoint: hello-hello-hello

  # This spec contains two templates: hello-hello-hello and print-message
  templates:
  - name: hello-hello-hello
    # Instead of just running a container
    # This template has a sequence of steps
    steps:
    - - name: hello1            # hello1 is run before the following steps
        template: print-message
        arguments:
          parameters:
          - name: message
            value: "hello1"
    - - name: hello2a           # double dash => run after previous step
        template: print-message
        arguments:
          parameters:
          - name: message
            value: "hello2a"
      - name: hello2b           # single dash => run in parallel with previous step
        template: print-message
        arguments:
          parameters:
          - name: message
            value: "hello2b"

  # This is the same template as from the previous example
  - name: print-message
    inputs:
      parameters:
      - name: message
    container:
      image: busybox
      command: [echo]
      args: ["{{inputs.parameters.message}}"]
```

///

/// tab | Python

```python
from hera.workflows import Container, Parameter, Step, Steps, Workflow # (1)!

with Workflow(
    generate_name="steps-",
    entrypoint="hello-hello-hello",
) as w:
    print_message = Container(
        name="print-message",
        inputs=[Parameter(name="message")],
        image="busybox",
        command=["echo"],
        args=["{{inputs.parameters.message}}"],
    )

    with Steps(name="hello-hello-hello") as s:
        Step(
            name="hello1",
            template=print_message,
            arguments=[Parameter(name="message", value="hello1")],
        )

        with s.parallel():
            Step(
                name="hello2a",
                template=print_message,
                arguments=[Parameter(name="message", value="hello2a")],
            )
            Step(
                name="hello2b",
                template=print_message,
                arguments=[Parameter(name="message", value="hello2b")],
            )
```

1. Install the `hera` package to define your Workflows in Python. Learn more at [the Hera docs](https://hera.readthedocs.io/en/stable/).

///

The above workflow prints three variants of "hello".
The `hello-hello-hello` template has three `steps`.
The first step, `hello1`, runs in sequence, whereas the next two steps, `hello2a` and `hello2b`, run in parallel with each other.

You can use the [`argo get` CLI command](../cli/argo_get.md) to display the execution history.
The example output below shows that `hello2a` and `hello2b` ran in parallel:

```bash
STEP            TEMPLATE           PODNAME                 DURATION  MESSAGE
 ✔ steps-z2zdn  hello-hello-hello
 ├───✔ hello1   print-message      steps-z2zdn-27420706    2s
 └─┬─✔ hello2a  print-message      steps-z2zdn-2006760091  3s
   └─✔ hello2b  print-message      steps-z2zdn-2023537710  3s
```
