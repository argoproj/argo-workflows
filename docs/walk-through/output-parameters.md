# Output Parameters

Output parameters provide a general mechanism to use the result of a step as a parameter (and not just as an artifact). This allows you to use the result from any type of step, not just a `script`, for conditional tests, loops, and arguments. Output parameters work similarly to `script result` except that the value of the output parameter is set to the contents of a generated file rather than the contents of `stdout`.

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
