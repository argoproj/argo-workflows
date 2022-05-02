# Steps

In this example, we'll see how to create multi-step workflows, how to define more than one template in a workflow spec, and how to create nested workflows. Be sure to read the comments as they provide useful explanations.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: steps-
spec:
  entrypoint: hello-hello-hello

  # This spec contains two templates: hello-hello-hello and whalesay
  templates:
  - name: hello-hello-hello
    # Instead of just running a container
    # This template has a sequence of steps
    steps:
    - - name: hello1            # hello1 is run before the following steps
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "hello1"
    - - name: hello2a           # double dash => run after previous step
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "hello2a"
      - name: hello2b           # single dash => run in parallel with previous step
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "hello2b"

  # This is the same template as from the previous example
  - name: whalesay
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
```

The above workflow spec prints three different flavors of "hello". The `hello-hello-hello` template consists of three `steps`. The first step named `hello1` will be run in sequence whereas the next two steps named `hello2a` and `hello2b` will be run in parallel with each other. Using the argo CLI command, we can graphically display the execution history of this workflow spec, which shows that the steps named `hello2a` and `hello2b` ran in parallel with each other.

```bash
STEP            TEMPLATE           PODNAME                 DURATION  MESSAGE
 ✔ steps-z2zdn  hello-hello-hello
 ├───✔ hello1   whalesay           steps-z2zdn-27420706    2s
 └─┬─✔ hello2a  whalesay           steps-z2zdn-2006760091  3s
   └─✔ hello2b  whalesay           steps-z2zdn-2023537710  3s
```
