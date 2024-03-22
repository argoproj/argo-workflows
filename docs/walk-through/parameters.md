# Parameters

Let's look at a slightly more complex workflow spec with parameters.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-parameters-
spec:
  # invoke the whalesay template with
  # "hello world" as the argument
  # to the message parameter
  entrypoint: whalesay
  arguments:
    parameters:
    - name: message
      value: hello world

  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: message       # parameter declaration
    container:
      # run cowsay with that message input parameter as args
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
```

This time, the `whalesay` template takes an input parameter named `message` that is passed as the `args` to the `cowsay` command. In order to reference parameters (e.g., ``"{{inputs.parameters.message}}"``), the parameters must be enclosed in double quotes to escape the curly braces in YAML.

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

Command-line parameters can also be used to override the default entrypoint and invoke any template in the workflow spec. For example, if you add a new version of the `whalesay` template called `whalesay-caps` but you don't want to change the default entrypoint, you can invoke this from the command line as follows:

```bash
argo submit arguments-parameters.yaml --entrypoint whalesay-caps
```

By using a combination of the `--entrypoint` and `-p` parameters, you can call any template in the workflow spec with any parameter that you like.

The values set in the `spec.arguments.parameters` are globally scoped and can be accessed via `{{workflow.parameters.parameter_name}}`. This can be useful to pass information to multiple steps in a workflow. For example, if you wanted to run your workflows with different logging levels that are set in the environment of each container, you could have a YAML file similar to this one:

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

In this workflow, both steps `A` and `B` would have the same log-level set to `INFO` and can easily be changed between workflow submissions using the `-p` flag.
