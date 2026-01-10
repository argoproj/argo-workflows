# Output Parameters

Output parameters allow multiple values to be outputted from any type of template, not just as an artifact or as the script `result` output.
Output parameters are declared within the template, similarly to artifacts, and also take their value from a file.
They can then be accessed from steps or tasks, and can be used for conditional tests, loops, and arguments.

/// tab | YAML

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: output-parameter-
spec:
  entrypoint: output-parameter
  templates:
  - name: output-parameter
    steps:
    - - name: generate-parameter
        template: hello-world-to-file
    - - name: consume-parameter
        template: print-message
        arguments:
          parameters:
          # Pass the hello-param output from the generate-parameter step as the message input to print-message
          - name: message
            value: "{{steps.generate-parameter.outputs.parameters.hello-param}}"

  - name: hello-world-to-file
    container:
      image: busybox
      command: [sh, -c]
      args: ["echo -n hello world > /tmp/hello_world.txt"]  # generate the content of hello_world.txt
    outputs:
      parameters:
      - name: hello-param  # name of output parameter
        valueFrom:
          path: /tmp/hello_world.txt # set the value of hello-param to the contents of this hello-world.txt

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
from hera.workflows import (
    Container,
    Parameter,
    Steps,
    Workflow,
    models as m,
)

with Workflow(generate_name="output-parameter-", entrypoint="output-parameter") as w:
    hello_world_to_file = Container(
        name="hello-world-to-file",
        image="busybox",
        command=["sh", "-c"],
        args=["echo -n hello world > /tmp/hello_world.txt"],
        outputs=Parameter(
            name="hello-param",
            value_from=m.ValueFrom(
                path="/tmp/hello_world.txt",
            ),
        ),
    )
    print_message = Container(
        name="print-message",
        image="busybox",
        command=["echo"],
        args=["{{inputs.parameters.message}}"],
        inputs=Parameter(name="message"),
    )
    with Steps(name="output-parameter"):
        generate_step = hello_world_to_file(name="generate-parameter")
        print_message(
            name="consume-parameter",
            arguments={"message": generate_step.get_parameter("hello-param")},
        )
```

///

DAG templates use the tasks prefix to refer to another task, for example `{{tasks.generate-parameter.outputs.parameters.hello-param}}`.

## `result` output parameter

For script and container templates, the `result` output parameter captures up to 256 kb of the standard output.
For HTTP templates, `result` captures the response body.
It is accessible from the `outputs` map: `outputs.result`.

### Scripts

Outputs of a `script` are assigned to standard output and captured in the `result` parameter. More details [here](scripts-and-results.md).

### Containers

Container steps and tasks also have their standard output captured in the `result` parameter.
Given a `task`, called `log-int`, `result` would then be accessible as `{{ tasks.log-int.outputs.result }}`. If using [steps](steps.md), substitute `tasks` for `steps`: `{{ steps.log-int.outputs.result }}`.

### HTTP

[HTTP templates](../http-template.md) capture the response body in the `result` parameter if the body is non-empty.
