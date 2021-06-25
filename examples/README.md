# Documentation by Example

## Welcome!

Argo is an open source project that provides container-native workflows for Kubernetes. Each step in an Argo workflow is defined as a container.

Argo is implemented as a Kubernetes CRD (Custom Resource Definition). As a result, Argo workflows can be managed using `kubectl` and natively integrates with other Kubernetes services such as volumes, secrets, and RBAC. The new Argo software is light-weight and installs in under a minute, and provides complete workflow features including parameter substitution, artifacts, fixtures, loops and recursive workflows.

Many of the Argo examples used in this walkthrough are available in the [`/examples` directory](https://github.com/argoproj/argo-workflows/tree/master/examples) on GitHub. If you like this project, please give us a star!

For a complete description of the Argo workflow spec, please refer to [our spec definitions](https://github.com/argoproj/argo-workflows/blob/master/pkg/apis/workflow/v1alpha1/workflow_types.go).

## Table of Contents

1. [Argo CLI](#argo-cli)
1. [Hello World!](#hello-world)
1. [Parameters](#parameters)
1. [Steps](#steps)
1. [DAG](#dag)
1. [Artifacts](#artifacts)
1. [The Structure of Workflow Specs](#the-structure-of-workflow-specs)
1. [Secrets](#secrets)
1. [Scripts & Results](#scripts--results)
1. [Output Parameters](#output-parameters)
1. [Loops](#loops)
1. [Conditionals](#conditionals)
1. [Retrying Failed or Errored Steps](#retrying-failed-or-errored-steps)
1. [Recursion](#recursion)
1. [Exit Handlers](#exit-handlers)
1. [Timeouts](#timeouts)
1. [Volumes](#volumes)
1. [Suspending](#suspending)
1. [Daemon Containers](#daemon-containers)
1. [Sidecars](#sidecars)
1. [Hardwired Artifacts](#hardwired-artifacts)
1. [Kubernetes Resources](#kubernetes-resources)
1. [Docker-in-Docker Using Sidecars](#docker-in-docker-using-sidecars)
1. [Custom Template Variable Reference](#custom-template-variable-reference)
1. [Continuous Integration Example](#continuous-integration-example)

## Argo CLI

In case you want to follow along with this walkthrough, here's a quick overview of the most useful argo command line interface (CLI) commands.

```sh
argo submit hello-world.yaml    # submit a workflow spec to Kubernetes
argo list                       # list current workflows
argo get hello-world-xxx        # get info about a specific workflow
argo logs hello-world-xxx       # print the logs from a workflow
argo delete hello-world-xxx     # delete workflow
```

You can also run workflow specs directly using `kubectl` but the Argo CLI provides syntax checking, nicer output, and requires less typing.

```sh
kubectl create -f hello-world.yaml
kubectl get wf
kubectl get wf hello-world-xxx
kubectl get po --selector=workflows.argoproj.io/workflow=hello-world-xxx --show-all  # similar to argo
kubectl logs hello-world-xxx-yyy -c main
kubectl delete wf hello-world-xxx
```

## Hello World!

Let's start by creating a very simple workflow template to echo "hello world" using the docker/whalesay container image from DockerHub.

You can run this directly from your shell with a simple docker command:

```sh
$ docker run docker/whalesay cowsay "hello world"
 _____________
< hello world >
 -------------
    \
     \
      \
                    ##        .
              ## ## ##       ==
           ## ## ## ##      ===
       /""""""""""""""""___/ ===
  ~~~ {~~ ~~~~ ~~~ ~~~~ ~~ ~ /  ===- ~~~
       \______ o          __/
        \    \        __/
          \____\______/


Hello from Docker!
This message shows that your installation appears to be working correctly.
```

Below, we run the same container on a Kubernetes cluster using an Argo workflow template.
Be sure to read the comments as they provide useful explanations.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow                  # new type of k8s spec
metadata:
  generateName: hello-world-    # name of the workflow spec
spec:
  entrypoint: whalesay          # invoke the whalesay template
  templates:
  - name: whalesay              # name of the template
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["hello world"]
      resources:                # limit the resources
        limits:
          memory: 32Mi
          cpu: 100m
```

Argo adds a new `kind` of Kubernetes spec called a `Workflow`. The above spec contains a single `template` called `whalesay` which runs the `docker/whalesay` container and invokes `cowsay "hello world"`. The `whalesay` template is the `entrypoint` for the spec. The entrypoint specifies the initial template that should be invoked when the workflow spec is executed by Kubernetes. Being able to specify the entrypoint is more useful when there is more than one template defined in the Kubernetes workflow spec. :-)

## Parameters

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

```sh
argo submit arguments-parameters.yaml -p message="goodbye world"
```

In case of multiple parameters that can be overriten, the argo CLI provides a command to load parameters files in YAML or JSON format. Here is an example of that kind of parameter file:

```yaml
message: goodbye world
```

To run use following command:

```sh
argo submit arguments-parameters.yaml --parameter-file params.yaml
```

Command-line parameters can also be used to override the default entrypoint and invoke any template in the workflow spec. For example, if you add a new version of the `whalesay` template called `whalesay-caps` but you don't want to change the default entrypoint, you can invoke this from the command line as follows:

```sh
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

## Steps

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

```sh
STEP            TEMPLATE           PODNAME                 DURATION  MESSAGE
 ✔ steps-z2zdn  hello-hello-hello
 ├───✔ hello1   whalesay           steps-z2zdn-27420706    2s
 └─┬─✔ hello2a  whalesay           steps-z2zdn-2006760091  3s
   └─✔ hello2b  whalesay           steps-z2zdn-2023537710  3s
```

## DAG

As an alternative to specifying sequences of steps, you can define the workflow as a directed-acyclic graph (DAG) by specifying the dependencies of each task. This can be simpler to maintain for complex workflows and allows for maximum parallelism when running tasks.

In the following workflow, step `A` runs first, as it has no dependencies. Once `A` has finished, steps `B` and `C` run in parallel. Finally, once `B` and `C` have completed, step `D` can run.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-diamond-
spec:
  entrypoint: diamond
  templates:
  - name: echo
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:3.7
      command: [echo, "{{inputs.parameters.message}}"]
  - name: diamond
    dag:
      tasks:
      - name: A
        template: echo
        arguments:
          parameters: [{name: message, value: A}]
      - name: B
        dependencies: [A]
        template: echo
        arguments:
          parameters: [{name: message, value: B}]
      - name: C
        dependencies: [A]
        template: echo
        arguments:
          parameters: [{name: message, value: C}]
      - name: D
        dependencies: [B, C]
        template: echo
        arguments:
          parameters: [{name: message, value: D}]
```

The dependency graph may have [multiple roots](./dag-multiroot.yaml). The templates called from a DAG or steps template can themselves be DAG or steps templates. This can allow for complex workflows to be split into manageable pieces.

The DAG logic has a built-in `fail fast` feature to stop scheduling new steps, as soon as it detects that one of the DAG nodes is failed. Then it waits until all DAG nodes are completed before failing the DAG itself.
The [FailFast](./dag-disable-failFast.yaml) flag default is `true`,  if set to `false`, it will allow a DAG to run all branches of the DAG to completion (either success or failure), regardless of the failed outcomes of branches in the DAG. More info and example about this feature at [here](https://github.com/argoproj/argo-workflows/issues/1442).
## Artifacts

**Note:**
You will need to configure an artifact repository to run this example.
[Configuring an artifact repository here](https://argoproj.github.io/argo-workflows/configure-artifact-repository/).

When running workflows, it is very common to have steps that generate or consume artifacts. Often, the output artifacts of one step may be used as input artifacts to a subsequent step.

The below workflow spec consists of two steps that run in sequence. The first step named `generate-artifact` will generate an artifact using the `whalesay` template that will be consumed by the second step named `print-message` that then consumes the generated artifact.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: artifact-passing-
spec:
  entrypoint: artifact-example
  templates:
  - name: artifact-example
    steps:
    - - name: generate-artifact
        template: whalesay
    - - name: consume-artifact
        template: print-message
        arguments:
          artifacts:
          # bind message to the hello-art artifact
          # generated by the generate-artifact step
          - name: message
            from: "{{steps.generate-artifact.outputs.artifacts.hello-art}}"

  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay hello world | tee /tmp/hello_world.txt"]
    outputs:
      artifacts:
      # generate hello-art artifact from /tmp/hello_world.txt
      # artifacts can be directories as well as files
      - name: hello-art
        path: /tmp/hello_world.txt

  - name: print-message
    inputs:
      artifacts:
      # unpack the message input artifact
      # and put it at /tmp/message
      - name: message
        path: /tmp/message
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["cat /tmp/message"]
```

The `whalesay` template uses the `cowsay` command to generate a file named `/tmp/hello-world.txt`. It then `outputs` this file as an artifact named `hello-art`. In general, the artifact's `path` may be a directory rather than just a file. The `print-message` template takes an input artifact named `message`, unpacks it at the `path` named `/tmp/message` and then prints the contents of `/tmp/message` using the `cat` command.
The `artifact-example` template passes the `hello-art` artifact generated as an output of the `generate-artifact` step as the `message` input artifact to the `print-message` step. DAG templates use the tasks prefix to refer to another task, for example `{{tasks.generate-artifact.outputs.artifacts.hello-art}}`.

Artifacts are packaged as Tarballs and gzipped by default. You may customize this behavior by specifying an archive strategy, using the `archive` field. For example:

```yaml
<... snipped ...>
    outputs:
      artifacts:
        # default behavior - tar+gzip default compression.
      - name: hello-art-1
        path: /tmp/hello_world.txt

        # disable archiving entirely - upload the file / directory as is.
        # this is useful when the container layout matches the desired target repository layout.   
      - name: hello-art-2
        path: /tmp/hello_world.txt
        archive:
          none: {}

        # customize the compression behavior (disabling it here).
        # this is useful for files with varying compression benefits, 
        # e.g. disabling compression for a cached build workspace and large binaries, 
        # or increasing compression for "perfect" textual data - like a json/xml export of a large database.
      - name: hello-art-3
        path: /tmp/hello_world.txt
        archive:
          tar:
            # no compression (also accepts the standard gzip 1 to 9 values)
            compressionLevel: 0
<... snipped ...>
``` 

## The Structure of Workflow Specs

We now know enough about the basic components of a workflow spec to review its basic structure:

- Kubernetes header including metadata
- Spec body
  - Entrypoint invocation with optionally arguments
  - List of template definitions

- For each template definition
  - Name of the template
  - Optionally a list of inputs
  - Optionally a list of outputs
  - Container invocation (leaf template) or a list of steps
    - For each step, a template invocation

To summarize, workflow specs are composed of a set of Argo templates where each template consists of an optional input section, an optional output section and either a container invocation or a list of steps where each step invokes another template.

Note that the container section of the workflow spec will accept the same options as the container section of a pod spec, including but not limited to environment variables, secrets, and volume mounts. Similarly, for volume claims and volumes.

## Secrets

Argo supports the same secrets syntax and mechanisms as Kubernetes Pod specs, which allows access to secrets as environment variables or volume mounts. See the [Kubernetes documentation](https://kubernetes.io/docs/concepts/configuration/secret/) for more information.

```yaml
# To run this example, first create the secret by running:
# kubectl create secret generic my-secret --from-literal=mypassword=S00perS3cretPa55word
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: secret-example-
spec:
  entrypoint: whalesay
  # To access secrets as files, add a volume entry in spec.volumes[] and
  # then in the container template spec, add a mount using volumeMounts.
  volumes:
  - name: my-secret-vol
    secret:
      secretName: my-secret     # name of an existing k8s secret
  templates:
  - name: whalesay
    container:
      image: alpine:3.7
      command: [sh, -c]
      args: ['
        echo "secret from env: $MYSECRETPASSWORD";
        echo "secret from file: `cat /secret/mountpath/mypassword`"
      ']
      # To access secrets as environment variables, use the k8s valueFrom and
      # secretKeyRef constructs.
      env:
      - name: MYSECRETPASSWORD  # name of env var
        valueFrom:
          secretKeyRef:
            name: my-secret     # name of an existing k8s secret
            key: mypassword     # 'key' subcomponent of the secret
      volumeMounts:
      - name: my-secret-vol     # mount file containing secret at /secret/mountpath
        mountPath: "/secret/mountpath"
```

## Scripts & Results

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

The `script` keyword allows the specification of the script body using the `source` tag. This creates a temporary file containing the script body and then passes the name of the temporary file as the final parameter to `command`, which should be an interpreter that executes the script body.

The use of the `script` feature also assigns the standard output of running the script to a special output parameter named `result`. This allows you to use the result of running the script itself in the rest of the workflow spec. In this example, the result is simply echoed by the print-message template.

## Output Parameters

Output parameters provide a general mechanism to use the result of a step as a parameter rather than as an artifact. This allows you to use the result from any type of step, not just a `script`, for conditional tests, loops, and arguments. Output parameters work similarly to `script result` except that the value of the output parameter is set to the contents of a generated file rather than the contents of `stdout`.

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
        template: whalesay
    - - name: consume-parameter
        template: print-message
        arguments:
          parameters:
          # Pass the hello-param output from the generate-parameter step as the message input to print-message
          - name: message
            value: "{{steps.generate-parameter.outputs.parameters.hello-param}}"

  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["echo -n hello world > /tmp/hello_world.txt"]  # generate the content of hello_world.txt
    outputs:
      parameters:
      - name: hello-param		# name of output parameter
        valueFrom:
          path: /tmp/hello_world.txt	# set the value of hello-param to the contents of this hello-world.txt

  - name: print-message
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
```

DAG templates use the tasks prefix to refer to another task, for example `{{tasks.generate-parameter.outputs.parameters.hello-param}}`.

## Loops

When writing workflows, it is often very useful to be able to iterate over a set of inputs as shown in this example:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: loops-
spec:
  entrypoint: loop-example
  templates:
  - name: loop-example
    steps:
    - - name: print-message
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "{{item}}"
        withItems:              # invoke whalesay once for each item in parallel
        - hello world           # item 1
        - goodbye world         # item 2

  - name: whalesay
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
```

We can also iterate over sets of items:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: loops-maps-
spec:
  entrypoint: loop-map-example
  templates:
  - name: loop-map-example
    steps:
    - - name: test-linux
        template: cat-os-release
        arguments:
          parameters:
          - name: image
            value: "{{item.image}}"
          - name: tag
            value: "{{item.tag}}"
        withItems:
        - { image: 'debian', tag: '9.1' }       #item set 1
        - { image: 'debian', tag: '8.9' }       #item set 2
        - { image: 'alpine', tag: '3.6' }       #item set 3
        - { image: 'ubuntu', tag: '17.10' }     #item set 4

  - name: cat-os-release
    inputs:
      parameters:
      - name: image
      - name: tag
    container:
      image: "{{inputs.parameters.image}}:{{inputs.parameters.tag}}"
      command: [cat]
      args: [/etc/os-release]
```

We can pass lists of items as parameters:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: loops-param-arg-
spec:
  entrypoint: loop-param-arg-example
  arguments:
    parameters:
    - name: os-list                                     # a list of items
      value: |
        [
          { "image": "debian", "tag": "9.1" },
          { "image": "debian", "tag": "8.9" },
          { "image": "alpine", "tag": "3.6" },
          { "image": "ubuntu", "tag": "17.10" }
        ]

  templates:
  - name: loop-param-arg-example
    inputs:
      parameters:
      - name: os-list
    steps:
    - - name: test-linux
        template: cat-os-release
        arguments:
          parameters:
          - name: image
            value: "{{item.image}}"
          - name: tag
            value: "{{item.tag}}"
        withParam: "{{inputs.parameters.os-list}}"      # parameter specifies the list to iterate over

  # This template is the same as in the previous example
  - name: cat-os-release
    inputs:
      parameters:
      - name: image
      - name: tag
    container:
      image: "{{inputs.parameters.image}}:{{inputs.parameters.tag}}"
      command: [cat]
      args: [/etc/os-release]
```

We can even dynamically generate the list of items to iterate over!

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: loops-param-result-
spec:
  entrypoint: loop-param-result-example
  templates:
  - name: loop-param-result-example
    steps:
    - - name: generate
        template: gen-number-list
    # Iterate over the list of numbers generated by the generate step above
    - - name: sleep
        template: sleep-n-sec
        arguments:
          parameters:
          - name: seconds
            value: "{{item}}"
        withParam: "{{steps.generate.outputs.result}}"

  # Generate a list of numbers in JSON format
  - name: gen-number-list
    script:
      image: python:alpine3.6
      command: [python]
      source: |
        import json
        import sys
        json.dump([i for i in range(20, 31)], sys.stdout)

  - name: sleep-n-sec
    inputs:
      parameters:
      - name: seconds
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo sleeping for {{inputs.parameters.seconds}} seconds; sleep {{inputs.parameters.seconds}}; echo done"]
```

## Conditionals

We also support conditional execution. The syntax is implemented by [govaluate](https://github.com/Knetic/govaluate) which offers the support for complex syntax. See in the example:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: coinflip-
spec:
  entrypoint: coinflip
  templates:
  - name: coinflip
    steps:
    # flip a coin
    - - name: flip-coin
        template: flip-coin
    # evaluate the result in parallel
    - - name: heads
        template: heads                       # call heads template if "heads"
        when: "{{steps.flip-coin.outputs.result}} == heads"
      - name: tails
        template: tails                       # call tails template if "tails"
        when: "{{steps.flip-coin.outputs.result}} == tails"
    - - name: flip-again
        template: flip-coin
    - - name: complex-condition
        template: heads-tails-or-twice-tails
        # call heads template if first flip was "heads" and second was "tails" OR both were "tails"
        when: >-
            ( {{steps.flip-coin.outputs.result}} == heads &&
              {{steps.flip-again.outputs.result}} == tails
            ) ||
            ( {{steps.flip-coin.outputs.result}} == tails &&
              {{steps.flip-again.outputs.result}} == tails )
      - name: heads-regex
        template: heads                       # call heads template if ~ "hea"
        when: "{{steps.flip-again.outputs.result}} =~ hea"
      - name: tails-regex
        template: tails                       # call heads template if ~ "tai"
        when: "{{steps.flip-again.outputs.result}} =~ tai"

  # Return heads or tails based on a random number
  - name: flip-coin
    script:
      image: python:alpine3.6
      command: [python]
      source: |
        import random
        result = "heads" if random.randint(0,1) == 0 else "tails"
        print(result)

  - name: heads
    container:
      image: alpine:3.6
      command: [sh, -c]
      args: ["echo \"it was heads\""]

  - name: tails
    container:
      image: alpine:3.6
      command: [sh, -c]
      args: ["echo \"it was tails\""]
  
  - name: heads-tails-or-twice-tails
    container:
      image: alpine:3.6
      command: [sh, -c]
      args: ["echo \"it was heads the first flip and tails the second. Or it was two times tails.\""]
```

## Retrying Failed or Errored Steps

You can specify a `retryStrategy` that will dictate how failed or errored steps are retried:

```yaml
# This example demonstrates the use of retry back offs
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: retry-backoff-
spec:
  entrypoint: retry-backoff
  templates:
  - name: retry-backoff
    retryStrategy:
      limit: 10
      retryPolicy: "Always"
      backoff:
        duration: "1"      # Must be a string. Default unit is seconds. Could also be a Duration, e.g.: "2m", "6h", "1d"
        factor: 2
        maxDuration: "1m"  # Must be a string. Default unit is seconds. Could also be a Duration, e.g.: "2m", "6h", "1d"
      affinity:
        nodeAntiAffinity: {}
    container:
      image: python:alpine3.6
      command: ["python", -c]
      # fail with a 66% probability
      args: ["import random; import sys; exit_code = random.choice([0, 1, 1]); sys.exit(exit_code)"]
```

* `limit` is the maximum number of times the container will be retried.
* `retryPolicy` specifies if a container will be retried on failure, error, both, or only transient errors (e.g. i/o or TLS handshake timeout). "Always" retries on both errors and failures. Also available: "OnFailure" (default), "OnError", and "OnTransientError" (available after v3.0.0-rc2).
* `backoff` is an exponential backoff
* `nodeAntiAffinity` prevents running steps on the same host.  Current implementation allows only empty `nodeAntiAffinity` (i.e. `nodeAntiAffinity: {}`) and by default it uses label `kubernetes.io/hostname` as the selector.

Providing an empty `retryStrategy` (i.e. `retryStrategy: {}`) will cause a container to retry until completion.


## Recursion

Templates can recursively invoke each other! In this variation of the above coin-flip template, we continue to flip coins until it comes up heads.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: coinflip-recursive-
spec:
  entrypoint: coinflip
  templates:
  - name: coinflip
    steps:
    # flip a coin
    - - name: flip-coin
        template: flip-coin
    # evaluate the result in parallel
    - - name: heads
        template: heads                 # call heads template if "heads"
        when: "{{steps.flip-coin.outputs.result}} == heads"
      - name: tails                     # keep flipping coins if "tails"
        template: coinflip
        when: "{{steps.flip-coin.outputs.result}} == tails"

  - name: flip-coin
    script:
      image: python:alpine3.6
      command: [python]
      source: |
        import random
        result = "heads" if random.randint(0,1) == 0 else "tails"
        print(result)

  - name: heads
    container:
      image: alpine:3.6
      command: [sh, -c]
      args: ["echo \"it was heads\""]
```

Here's the result of a couple of runs of coinflip for comparison.

```sh
argo get coinflip-recursive-tzcb5

STEP                         PODNAME                              MESSAGE
 ✔ coinflip-recursive-vhph5
 ├───✔ flip-coin             coinflip-recursive-vhph5-2123890397
 └─┬─✔ heads                 coinflip-recursive-vhph5-128690560
   └─○ tails

STEP                          PODNAME                              MESSAGE
 ✔ coinflip-recursive-tzcb5
 ├───✔ flip-coin              coinflip-recursive-tzcb5-322836820
 └─┬─○ heads
   └─✔ tails
     ├───✔ flip-coin          coinflip-recursive-tzcb5-1863890320
     └─┬─○ heads
       └─✔ tails
         ├───✔ flip-coin      coinflip-recursive-tzcb5-1768147140
         └─┬─○ heads
           └─✔ tails
             ├───✔ flip-coin  coinflip-recursive-tzcb5-4080411136
             └─┬─✔ heads      coinflip-recursive-tzcb5-4080323273
               └─○ tails
```

In the first run, the coin immediately comes up heads and we stop. In the second run, the coin comes up tail three times before it finally comes up heads and we stop.

## Exit handlers

An exit handler is a template that *always* executes, irrespective of success or failure, at the end of the workflow.

Some common use cases of exit handlers are:

- cleaning up after a workflow runs
- sending notifications of workflow status (e.g., e-mail/Slack)
- posting the pass/fail status to a webhook result (e.g. GitHub build result)
- resubmitting or submitting another workflow

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: exit-handlers-
spec:
  entrypoint: intentional-fail
  onExit: exit-handler                  # invoke exit-handler template at end of the workflow
  templates:
  # primary workflow template
  - name: intentional-fail
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo intentional failure; exit 1"]

  # Exit handler templates
  # After the completion of the entrypoint template, the status of the
  # workflow is made available in the global variable {{workflow.status}}.
  # {{workflow.status}} will be one of: Succeeded, Failed, Error
  - name: exit-handler
    steps:
    - - name: notify
        template: send-email
      - name: celebrate
        template: celebrate
        when: "{{workflow.status}} == Succeeded"
      - name: cry
        template: cry
        when: "{{workflow.status}} != Succeeded"
  - name: send-email
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo send e-mail: {{workflow.name}} {{workflow.status}}"]
  - name: celebrate
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo hooray!"]
  - name: cry
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo boohoo!"]
```

## Timeouts

To limit the elapsed time for a workflow, you can set the variable `activeDeadlineSeconds`.

```yaml
# To enforce a timeout for a container template, specify a value for activeDeadlineSeconds.
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: timeouts-
spec:
  entrypoint: sleep
  templates:
  - name: sleep
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo sleeping for 1m; sleep 60; echo done"]
    activeDeadlineSeconds: 10           # terminate container template after 10 seconds
```

## Volumes

The following example dynamically creates a volume and then uses the volume in a two step workflow.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: volumes-pvc-
spec:
  entrypoint: volumes-pvc-example
  volumeClaimTemplates:                 # define volume, same syntax as k8s Pod spec
  - metadata:
      name: workdir                     # name of volume claim
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 1Gi                  # Gi => 1024 * 1024 * 1024

  templates:
  - name: volumes-pvc-example
    steps:
    - - name: generate
        template: whalesay
    - - name: print
        template: print-message

  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["echo generating message in volume; cowsay hello world | tee /mnt/vol/hello_world.txt"]
      # Mount workdir volume at /mnt/vol before invoking docker/whalesay
      volumeMounts:                     # same syntax as k8s Pod spec
      - name: workdir
        mountPath: /mnt/vol

  - name: print-message
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo getting message from volume; find /mnt/vol; cat /mnt/vol/hello_world.txt"]
      # Mount workdir volume at /mnt/vol before invoking docker/whalesay
      volumeMounts:                     # same syntax as k8s Pod spec
      - name: workdir
        mountPath: /mnt/vol

```

Volumes are a very useful way to move large amounts of data from one step in a workflow to another. Depending on the system, some volumes may be accessible concurrently from multiple steps.

In some cases, you want to access an already existing volume rather than creating/destroying one dynamically.

```yaml
# Define Kubernetes PVC
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: my-existing-volume
spec:
  accessModes: [ "ReadWriteOnce" ]
  resources:
    requests:
      storage: 1Gi

---
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: volumes-existing-
spec:
  entrypoint: volumes-existing-example
  volumes:
  # Pass my-existing-volume as an argument to the volumes-existing-example template
  # Same syntax as k8s Pod spec
  - name: workdir
    persistentVolumeClaim:
      claimName: my-existing-volume

  templates:
  - name: volumes-existing-example
    steps:
    - - name: generate
        template: whalesay
    - - name: print
        template: print-message

  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["echo generating message in volume; cowsay hello world | tee /mnt/vol/hello_world.txt"]
      volumeMounts:
      - name: workdir
        mountPath: /mnt/vol

  - name: print-message
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo getting message from volume; find /mnt/vol; cat /mnt/vol/hello_world.txt"]
      volumeMounts:
      - name: workdir
        mountPath: /mnt/vol
```

It's also possible to declare existing volumes at the template level, instead of the workflow level.
This can be useful workflows that generate volumes using a `resource` step.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: template-level-volume-
spec:
  entrypoint: generate-and-use-volume
  templates:
  - name: generate-and-use-volume
    steps:
    - - name: generate-volume
        template: generate-volume
        arguments:
          parameters:
            - name: pvc-size
              # In a real-world example, this could be generated by a previous workflow step.
              value: '1Gi'
    - - name: generate
        template: whalesay
        arguments:
          parameters:
            - name: pvc-name
              value: '{{steps.generate-volume.outputs.parameters.pvc-name}}'
    - - name: print
        template: print-message
        arguments:
          parameters:
            - name: pvc-name
              value: '{{steps.generate-volume.outputs.parameters.pvc-name}}'

  - name: generate-volume
    inputs:
      parameters:
        - name: pvc-size
    resource:
      action: create
      setOwnerReference: true
      manifest: |
        apiVersion: v1
        kind: PersistentVolumeClaim
        metadata:
          generateName: pvc-example-
        spec:
          accessModes: ['ReadWriteOnce', 'ReadOnlyMany']
          resources:
            requests:
              storage: '{{inputs.parameters.pvc-size}}'
    outputs:
      parameters:
        - name: pvc-name
          valueFrom:
            jsonPath: '{.metadata.name}'

  - name: whalesay
    inputs:
      parameters:
        - name: pvc-name
    volumes:
      - name: workdir
        persistentVolumeClaim:
          claimName: '{{inputs.parameters.pvc-name}}'
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["echo generating message in volume; cowsay hello world | tee /mnt/vol/hello_world.txt"]
      volumeMounts:
      - name: workdir
        mountPath: /mnt/vol

  - name: print-message
    inputs:
        parameters:
          - name: pvc-name
    volumes:
      - name: workdir
        persistentVolumeClaim:
          claimName: '{{inputs.parameters.pvc-name}}'
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo getting message from volume; find /mnt/vol; cat /mnt/vol/hello_world.txt"]
      volumeMounts:
      - name: workdir
        mountPath: /mnt/vol

```

## Suspending

Workflows can be suspended by

```sh
argo suspend WORKFLOW
```

Or by specifying a `suspend` step on the workflow:

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
        template: whalesay
    - - name: approve
        template: approve
    - - name: delay
        template: delay
    - - name: release
        template: whalesay

  - name: approve
    suspend: {}

  - name: delay
    suspend:
      duration: "20"    # Must be a string. Default unit is seconds. Could also be a Duration, e.g.: "2m", "6h", "1d"

  - name: whalesay
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["hello world"]
```

Once suspended, a Workflow will not schedule any new steps until it is resumed. It can be resumed manually by
```sh
argo resume WORKFLOW
```
Or automatically with a `duration` limit as the example above.

## Daemon Containers

Argo workflows can start containers that run in the background (also known as `daemon containers`) while the workflow itself continues execution. Note that the daemons will be *automatically destroyed* when the workflow exits the template scope in which the daemon was invoked. Daemon containers are useful for starting up services to be tested or to be used in testing (e.g., fixtures). We also find it very useful when running large simulations to spin up a database as a daemon for collecting and organizing the results. The big advantage of daemons compared with sidecars is that their existence can persist across multiple steps or even the entire workflow.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: daemon-step-
spec:
  entrypoint: daemon-example
  templates:
  - name: daemon-example
    steps:
    - - name: influx
        template: influxdb              # start an influxdb as a daemon (see the influxdb template spec below)

    - - name: init-database             # initialize influxdb
        template: influxdb-client
        arguments:
          parameters:
          - name: cmd
            value: curl -XPOST 'http://{{steps.influx.ip}}:8086/query' --data-urlencode "q=CREATE DATABASE mydb"

    - - name: producer-1                # add entries to influxdb
        template: influxdb-client
        arguments:
          parameters:
          - name: cmd
            value: for i in $(seq 1 20); do curl -XPOST 'http://{{steps.influx.ip}}:8086/write?db=mydb' -d "cpu,host=server01,region=uswest load=$i" ; sleep .5 ; done
      - name: producer-2                # add entries to influxdb
        template: influxdb-client
        arguments:
          parameters:
          - name: cmd
            value: for i in $(seq 1 20); do curl -XPOST 'http://{{steps.influx.ip}}:8086/write?db=mydb' -d "cpu,host=server02,region=uswest load=$((RANDOM % 100))" ; sleep .5 ; done
      - name: producer-3                # add entries to influxdb
        template: influxdb-client
        arguments:
          parameters:
          - name: cmd
            value: curl -XPOST 'http://{{steps.influx.ip}}:8086/write?db=mydb' -d 'cpu,host=server03,region=useast load=15.4'

    - - name: consumer                  # consume intries from influxdb
        template: influxdb-client
        arguments:
          parameters:
          - name: cmd
            value: curl --silent -G http://{{steps.influx.ip}}:8086/query?pretty=true --data-urlencode "db=mydb" --data-urlencode "q=SELECT * FROM cpu"

  - name: influxdb
    daemon: true                        # start influxdb as a daemon
    retryStrategy:
      limit: 10                         # retry container if it fails
    container:
      image: influxdb:1.2
      readinessProbe:                   # wait for readinessProbe to succeed
        httpGet:
          path: /ping
          port: 8086

  - name: influxdb-client
    inputs:
      parameters:
      - name: cmd
    container:
      image: appropriate/curl:latest
      command: ["/bin/sh", "-c"]
      args: ["{{inputs.parameters.cmd}}"]
      resources:
        requests:
          memory: 32Mi
          cpu: 100m
```

Step templates use the `steps` prefix to refer to another step: for example `{{steps.influx.ip}}`. In DAG templates, the `tasks` prefix is used instead: for example `{{tasks.influx.ip}}`.

## Sidecars

A sidecar is another container that executes concurrently in the same pod as the main container and is useful in creating multi-container pods.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: sidecar-nginx-
spec:
  entrypoint: sidecar-nginx-example
  templates:
  - name: sidecar-nginx-example
    container:
      image: appropriate/curl
      command: [sh, -c]
      # Try to read from nginx web server until it comes up
      args: ["until `curl -G 'http://127.0.0.1/' >& /tmp/out`; do echo sleep && sleep 1; done && cat /tmp/out"]
    # Create a simple nginx web server
    sidecars:
    - name: nginx
      image: nginx:1.13
```

In the above example, we create a sidecar container that runs nginx as a simple web server. The order in which containers come up is random, so in this example the main container polls the nginx container until it is ready to service requests. This is a good design pattern when designing multi-container systems: always wait for any services you need to come up before running your main code.

## Hardwired Artifacts

With Argo, you can use any container image that you like to generate any kind of artifact. In practice, however, we find certain types of artifacts are very common, so there is built-in support for git, http, gcs and s3 artifacts.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hardwired-artifact-
spec:
  entrypoint: hardwired-artifact
  templates:
  - name: hardwired-artifact
    inputs:
      artifacts:
      # Check out the master branch of the argo repo and place it at /src
      # revision can be anything that git checkout accepts: branch, commit, tag, etc.
      - name: argo-source
        path: /src
        git:
          repo: https://github.com/argoproj/argo-workflows.git
          revision: "master"
      # Download kubectl 1.8.0 and place it at /bin/kubectl
      - name: kubectl
        path: /bin/kubectl
        mode: 0755
        http:
          url: https://storage.googleapis.com/kubernetes-release/release/v1.8.0/bin/linux/amd64/kubectl
      # Copy an s3 compatible artifact repository bucket (such as AWS, GCS and Minio) and place it at /s3
      - name: objects
        path: /s3
        s3:
          endpoint: storage.googleapis.com
          bucket: my-bucket-name
          key: path/in/bucket
          accessKeySecret:
            name: my-s3-credentials
            key: accessKey
          secretKeySecret:
            name: my-s3-credentials
            key: secretKey
    container:
      image: debian
      command: [sh, -c]
      args: ["ls -l /src /bin/kubectl /s3"]
```

## Kubernetes Resources

In many cases, you will want to manage Kubernetes resources from Argo workflows. The resource template allows you to create, delete or updated any type of Kubernetes resource.

```yaml
# in a workflow. The resource template type accepts any k8s manifest
# (including CRDs) and can perform any kubectl action against it (e.g. create,
# apply, delete, patch).
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: k8s-jobs-
spec:
  entrypoint: pi-tmpl
  templates:
  - name: pi-tmpl
    resource:                   # indicates that this is a resource template
      action: create            # can be any kubectl action (e.g. create, delete, apply, patch)
      # The successCondition and failureCondition are optional expressions.
      # If failureCondition is true, the step is considered failed.
      # If successCondition is true, the step is considered successful.
      # They use kubernetes label selection syntax and can be applied against any field
      # of the resource (not just labels). Multiple AND conditions can be represented by comma
      # delimited expressions.
      # For more details: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
      successCondition: status.succeeded > 0
      failureCondition: status.failed > 3
      manifest: |               #put your kubernetes spec here
        apiVersion: batch/v1
        kind: Job
        metadata:
          generateName: pi-job-
        spec:
          template:
            metadata:
              name: pi
            spec:
              containers:
              - name: pi
                image: perl
                command: ["perl",  "-Mbignum=bpi", "-wle", "print bpi(2000)"]
              restartPolicy: Never
          backoffLimit: 4
```

**Note:**
Currently only a single resource can be managed by a resource template so either a `generateName` or `name` must be provided in the resource's metadata.

Resources created in this way are independent of the workflow. If you want the resource to be deleted when the workflow is deleted then you can use [Kubernetes garbage collection](https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/) with the workflow resource as an owner reference ([example](./k8s-owner-reference.yaml)).

You can also collect data about the resource in output parameters (see more at [k8s-jobs.yaml](./k8s-jobs.yaml))

**Note:**
When patching, the resource will accept another attribute, `mergeStrategy`, which can either be `strategic`, `merge`, or `json`. If this attribute is not supplied, it will default to `strategic`. Keep in mind that Custom Resources cannot be patched with `strategic`, so a different strategy must be chosen. For example, suppose you have the [CronTab CustomResourceDefinition](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#create-a-customresourcedefinition) defined, and the following instance of a CronTab:

```yaml
apiVersion: "stable.example.com/v1"
kind: CronTab
spec:
  cronSpec: "* * * * */5"
  image: my-awesome-cron-image
```

This Crontab can be modified using the following Argo Workflow:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: k8s-patch-
spec:
  entrypoint: cront-tmpl
  templates:
  - name: cront-tmpl
    resource:
      action: patch
      mergeStrategy: merge                 # Must be one of [strategic merge json]
      manifest: |
        apiVersion: "stable.example.com/v1"
        kind: CronTab
        spec:
          cronSpec: "* * * * */10"
          image: my-awesome-cron-image
```

## Docker-in-Docker Using Sidecars

An application of sidecars is to implement Docker-in-Docker (DinD). DinD is useful when you want to run Docker commands from inside a container. For example, you may want to build and push a container image from inside your build container. In the following example, we use the docker:dind container to run a Docker daemon in a sidecar and give the main container access to the daemon.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: sidecar-dind-
spec:
  entrypoint: dind-sidecar-example
  templates:
  - name: dind-sidecar-example
    container:
      image: docker:19.03.13
      command: [sh, -c]
      args: ["until docker ps; do sleep 3; done; docker run --rm debian:latest cat /etc/os-release"]
      env:
      - name: DOCKER_HOST               # the docker daemon can be access on the standard port on localhost
        value: 127.0.0.1
    sidecars:
    - name: dind
      image: docker:19.03.13-dind          # Docker already provides an image for running a Docker daemon
      env:
        - name: DOCKER_TLS_CERTDIR         # Docker TLS env config
          value: ""
      securityContext:
        privileged: true                # the Docker daemon can only run in a privileged container
      # mirrorVolumeMounts will mount the same volumes specified in the main container
      # to the sidecar (including artifacts), at the same mountPaths. This enables
      # dind daemon to (partially) see the same filesystem as the main container in
      # order to use features such as docker volume binding.
      mirrorVolumeMounts: true
```

## Custom Template Variable Reference

In this example, we can see how we can use the other template language variable reference (E.g: Jinja) in Argo workflow template.
Argo will validate and resolve only the variable that starts with Argo allowed prefix
{***"item", "steps", "inputs", "outputs", "workflow", "tasks"***}

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: custom-template-variable-
spec:
  entrypoint: hello-hello-hello

  templates:
    - name: hello-hello-hello
      steps:
        - - name: hello1
            template: whalesay
            arguments:
              parameters: [{name: message, value: "hello1"}]
        - - name: hello2a
            template: whalesay
            arguments:
              parameters: [{name: message, value: "hello2a"}]
          - name: hello2b
            template: whalesay
            arguments:
              parameters: [{name: message, value: "hello2b"}]

    - name: whalesay
      inputs:
        parameters:
          - name: message
      container:
        image: docker/whalesay
        command: [cowsay]
        args: ["{{user.username}}"]

```

## Continuous Integration Example

Continuous integration is a popular application for workflows. Currently, Argo does not provide event triggers for automatically kicking off your CI jobs, but we plan to do so in the near future. Until then, you can easily write a cron job that checks for new commits and kicks off the needed workflow, or use your existing Jenkins server to kick off the workflow.

A good example of a CI workflow spec is provided at https://github.com/argoproj/argo-workflows/tree/master/examples/influxdb-ci.yaml. Because it just uses the concepts that we've already covered and is somewhat long, we don't go into details here.
