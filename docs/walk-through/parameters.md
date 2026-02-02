# Parameters

Let's look at a slightly more complex workflow spec with parameters.

/// tab | YAML

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-parameters-
spec:
  entrypoint: print-message # (1)!
  arguments:
    parameters:
    - name: message
      value: hello world

  templates:
  - name: print-message
    inputs:
      parameters:
      - name: message # (2)!
    container: # (3)!
      image: busybox
      command: [echo]
      args: ["{{inputs.parameters.message}}"]
```

1. Invoke the print-message template with "hello world" as the argument to the message parameter.
2. This declares `message` as an input parameter.
3. This container runs `echo` with the `message` input parameter as `args`.

///

/// tab | Python

```python
from hera.workflows import Container, Parameter, Workflow # (1)!

with Workflow(
    generate_name="hello-world-parameters-",
    entrypoint="print-message",
    arguments=Parameter(name="message", value="hello world"),
) as w:
    Container(
        name="print-message",
        image="busybox",
        command=["echo"],
        args=["{{inputs.parameters.message}}"],
        inputs=Parameter(name="message"),
    )
```

1. Install the `hera` package to define your Workflows in Python. Learn more at [the Hera docs](https://hera.readthedocs.io/en/stable/).

///

This time, the `print-message` template takes an input parameter named `message` that is passed as the `args` to the `echo` command. In order to reference parameters (e.g., ``"{{inputs.parameters.message}}"``), the parameters must be enclosed in double quotes to escape the curly braces in YAML.

The argo CLI provides a convenient way to override parameters used to invoke the entrypoint. For example, the following command would bind the `message` parameter to "goodbye world" instead of the default "hello world".

```bash
argo submit arguments-parameters.yaml -p message="goodbye world"
```

In case of multiple parameters that can be overridden, the argo CLI provides a command to load parameters files in YAML or JSON format. Here is an example of that kind of parameter file:

```yaml
message: goodbye world
```

To run use following command:

```bash
argo submit arguments-parameters.yaml --parameter-file params.yaml
```

Command-line parameters can also be used to override the default entrypoint and invoke any template in the workflow spec. For example, if you add a new version of the `print-message` template called `print-message-caps` but you don't want to change the default entrypoint, you can invoke this from the command line as follows:

```bash
argo submit arguments-parameters.yaml --entrypoint print-message-caps
```

By using a combination of the `--entrypoint` and `-p` parameters, you can call any template in the workflow spec with any parameter that you like.

The values set in the `spec.arguments.parameters` are globally scoped and can be accessed via `{{workflow.parameters.parameter_name}}`. This can be useful to pass information to multiple steps in a workflow. For example, if you wanted to run your workflows with different logging levels that are set in the environment of each container, you could have a YAML file similar to this one:

/// tab | YAML

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: global-parameters-
spec:
  entrypoint: A
  arguments:
    parameters:
    - name: log-level
      value: INFO

  templates:
  - name: A
    container:
      image: containerA
      env:
      - name: LOG_LEVEL
        value: "{{workflow.parameters.log-level}}"
      command: [runA]
  - name: B
    container:
      image: containerB
      env:
      - name: LOG_LEVEL
        value: "{{workflow.parameters.log-level}}"
      command: [runB]
```

///

/// tab | Python

```python
from hera.workflows.models import Arguments, EnvVar, Parameter
from hera.workflows import Container, Workflow

with Workflow(
    generate_name="global-parameters-",
    entrypoint="A",
    arguments=[Parameter(name="log-level", value="INFO")],
) as w:
    Container(
        name="A",
        image="containerA",
        command=["runA"],
        env=[EnvVar(name="LOG_LEVEL", value="{{workflow.parameters.log-level}}")],
    )
    Container(
        name="B",
        image="containerB",
        command=["runB"],
        env=[EnvVar(name="LOG_LEVEL", value="{{workflow.parameters.log-level}}")],
    )
```

///

In this workflow, both steps `A` and `B` would have the same log-level set to `INFO` and can easily be changed between workflow submissions using the `-p` flag.
