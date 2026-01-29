# Scripts And Results

## Running Scripts Using `source`

Often, we just want a template that executes a script specified as a [here-script](https://en.wikipedia.org/wiki/Here_document) (also known as a `here document`) in the workflow spec.
You can pass source code to the `source` parameter of a `script` template to run it through the given `image` and `command`.
This example shows how to do that:

/// tab | YAML

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
      image: python:alpine3.23
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
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo result was: {{inputs.parameters.message}}"]
```

///

/// tab | Python

```python
from hera.workflows import Container, Parameter, Script, Steps, Workflow

with Workflow(
    generate_name="scripts-bash-",
    entrypoint="bash-script-example",
) as w:
    bash_script = Script(
        name="gen-random-int-bash",
        image="debian:9.4",
        command=["bash"],
        source="cat /dev/urandom | od -N2 -An -i | awk -v f=1 -v r=100 '{printf \"%i\\n\", f + r * $1 / 65536}'\n",
    )
    python_script = Script(
        name="gen-random-int-python",
        image="python:alpine3.6",
        command=["python"],
        source="import random\ni = random.randint(1, 100)\nprint(i)\n",
    )
    javascript_script = Script(
        name="gen-random-int-javascript",
        image="node:9.1-alpine",
        command=["node"],
        source="var rand = Math.floor(Math.random() * 100);\nconsole.log(rand);\n",
    )

    print_message = Container(
        name="print-message",
        image="alpine:latest",
        command=["sh", "-c"],
        args=["echo result was: {{inputs.parameters.message}}"],
        inputs=[Parameter(name="message")],
    )

    with Steps(name="bash-script-example") as steps:
        bash_script(name="generate")
        print_message(
            name="print",
            arguments={"message": "{{steps.generate.outputs.result}}"},
        )
```

///

The `script` keyword allows the specification of the script body using the `source` tag.
This creates a temporary file containing the script body and then passes the name of the temporary file as the final parameter to `command`, which should be an interpreter that executes the script body.

In the same way as `container` templates do, the use of the `script` feature also assigns the standard output of running the script to a special output parameter named `result`.
This allows you to use the result of running the script itself in the rest of the workflow spec.
In this example, the result is simply echoed by the print-message template.

## Python Scripts Using Hera

If you are mainly using Python Scripts, the Python SDK (Hera) offers a `script` decorator to turn any regular Python function into a Script Template.
Hera will convert input parameters of the function into input parameters for the Script Template, add boilerplate for `json` loading the values, and will then run the rest of your script verbatim.
As mentioned above, the contents of the `source` tag are copied into a temporary file, so your whole script, including imports and variables, must be defined within that function:

/// tab | Python

```py
from hera.workflows import Steps, Workflow, script


@script()
def roll_die(message: str):
    import random

    print(message)
    print("I'm rolling a die:", random.randint(1, 6))

@script()
def echo_result(message: str):
    print(message)

with Workflow(
    generate_name="python-script-",
    entrypoint="steps",
) as w:
    with Steps(name="steps"):
        roll_step = roll_die(arguments={"message": "Hello world!"})
        echo_result(arguments={"message": roll_step.result})

```

///

/// tab | YAML

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: python-script-
spec:
  entrypoint: steps
  templates:
  - name: steps
    steps:
    - - name: roll-die
        template: roll-die
        arguments:
          parameters:
          - name: message
            value: Hello world!
    - - name: echo-result
        template: echo-result
        arguments:
          parameters:
          - name: message
            value: '{{steps.roll-die.outputs.result}}'
  - name: roll-die
    inputs:
      parameters:
      - name: message
    script:
      image: python:3.10
      command:
      - python
      source: |-
        import os
        import sys
        sys.path.append(os.getcwd())
        import json
        try: message = json.loads(r'''{{inputs.parameters.message}}''')
        except: message = r'''{{inputs.parameters.message}}'''

        import random
        print(message)
        print("I'm rolling a die:", random.randint(1, 6))
  - name: echo-result
    inputs:
      parameters:
      - name: message
    script:
      image: python:3.10
      command:
      - python
      source: |-
        import os
        import sys
        sys.path.append(os.getcwd())
        import json
        try: message = json.loads(r'''{{inputs.parameters.message}}''')
        except: message = r'''{{inputs.parameters.message}}'''

        print(message)
```

///

Hera offers native Python integrations for Script Templates to avoid the pitfalls of the `source` tag and to improve input and output handling.
You will also be able to test functions as normal Python functions.
Read more in [the Hera Scripts guide](https://hera.readthedocs.io/en/stable/user-guides/script-basics/).
