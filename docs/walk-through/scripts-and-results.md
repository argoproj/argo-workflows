# Scripts And Results

Often, we just want a template that executes a script specified as a here-script (also known as a `here document`) in the workflow spec. This example shows how to do that:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: scripts-bash-
spec:
  entrypoint: bash-script-example
  templates:
  - name: bash-script-example
    steps:
    - - name: generate
        template: gen-random-int-bash
    - - name: print
        template: print-message
        arguments:
          parameters:
          - name: message
            value: "{{steps.generate.outputs.result}}"  # The result of the here-script

  - name: gen-random-int-bash
    script:
      image: debian:9.4
      command: [bash]
      source: |                                         # Contents of the here-script
        cat /dev/urandom | od -N2 -An -i | awk -v f=1 -v r=100 '{printf "%i\n", f + r * $1 / 65536}'

  - name: gen-random-int-python
    script:
      image: python:alpine3.6
      command: [python]
      source: |
        import random
        i = random.randint(1, 100)
        print(i)

  - name: gen-random-int-javascript
    script:
      image: node:9.1-alpine
      command: [node]
      source: |
        var rand = Math.floor(Math.random() * 100);
        console.log(rand);

  - name: print-message
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo result was: {{inputs.parameters.message}}"]
```

The `script` keyword allows the specification of the script body using the `source` tag.
This creates a temporary file containing the script body and then passes the name of the temporary file as the final parameter to `command`, which should be an interpreter that executes the script body.

In the same way as `container` tempalates do, the use of the `script` feature also assigns the standard output of running the script to a special output parameter named `result`.
This allows you to use the result of running the script itself in the rest of the workflow spec.
In this example, the result is simply echoed by the print-message template.
