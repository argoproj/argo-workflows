# Argo Workflow Templates by Example

## Welcome!

Argo is an open source project that provides container-native workflows for Kubernetes. Each step in an Argo workflow is defined as a container.

Argo is implemented as a Kubernetes CRD (Custom Resource Definition). As a result, Argo workflows can be managed using kubectl and natively integrates with other Kubernetes services such as volumes, secrets, and RBAC. The new Argo software is lightweight and installs in under a minute but provides complete workflow features including parameter substitution, artifacts, fixtures, loops and recursive workflows.

Many of the Argo examples used in this walkthrough are available at https://github.com/argoproj/argo/tree/master/examples.  If you like this project, please give us a star!

## Argo CLI

In case you want to follow along with this walkthrough, here's a quick overview of the most useful argo CLI commands.

[Install Argo here](https://github.com/argoproj/argo/blob/master/demo.md)

```
argo submit hello-world.yaml    #submit a workflow spec to Kubernetes
argo list                       #list current workflows
argo get hello-world-xxx        #get info about a specific workflow
argo logs hello-world-xxx-yyy   #get logs from a specific step in a workflow
argo delete hello-world-xxx     #delete workflow
```

You can also run workflow specs directly using kubectl but the argo cli provides syntax checking, nicer output, and requires less typing.
```
kubectl create -f hello-world.yaml
kubectl get wf
kubectl get wf hello-world-xxx
kubectl get po --selector=workflows.argoproj.io/workflow=hello-world-xxx --show-all  #similar to argo 
kubectl logs hello-world-xxx-yyy -c main
kubectl delete wf hello-world-xxx
```

## Hello World!

Let's start by creating a very simple workflow template to echo "hello world" using the docker/whalesay container image from DockerHub.

<img src="whalesay.png" width="600">

You can run this directly from your shell with a simple docker command.
```
bash% docker run docker/whalesay cowsay "hello world"
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
Be sure to read the comments. They provide useful explanations.
```
apiVersion: argoproj.io/v1alpha1
kind: Workflow                  #new type of k8s spec
metadata:
  generateName: hello-world-    #name of workflow spec
spec:
  entrypoint: whalesay          #invoke the whalesay template
  templates:
  - name: whalesay              #name of template
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["hello world"]
      resources:                #don't use too much resources
        limits:
          memory: 32Mi
          cpu: 100m
```
Argo adds a new `kind` of Kubernetes spec called a `Workflow`.
The above spec contains a single `template` called `whalesay` which runs the `docker/whalesay` container and invokes `cowsay "hello world"`.
The `whalesay` template is denoted as the `entrypoint` for the spec. The entrypoint specifies the initial template that should be invoked  when the workflow spec is executed by Kubernetes. Being able to specify the entrypoint is more useful when there are more than one template defined in the Kubernetes workflow spec :-)

## Parameters

Let's look at a slightly more complex workflow spec with parameters.
```
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
      - name: message       #parameter declaration
    container:
      # run cowsay with that message input parameter as args
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
```
This time, the `whalesay` template takes an input parameter named `message` which is passed as the `args` to the `cowsay` command. In order to reference parameters (e.g. "{{inputs.parameters.message}}"), the parameters must be enclosed in double quotes to escape the curly braces in YAML.

The argo CLI provides a convenient way to override parameters used to invoke the entrypoint. For example, the following command would bind the `message` parameter to "goodbye world" instead of the default "hello world".
```
argo submit arguments-parameters.yaml -p message="goodbye world"
```

Command line parameters can also be used to override the default entrypoint and invoke any template in the workflow spec. For example, if you add a new version of the `whalesay` template called `whalesay-caps` but you don't want to change the default entrypoint, you can invoke this from the command line as follows.
```
argo submit arguments-parameters.yaml --entrypoint whalesay2
```

By using a combination of the `--entrypoint` and `-p` parameters, you can invoke any template in the workflow spec with any parameter that you like.

## Steps

In this example, we'll see how to create multi-step workflows as well as how to define more than one template in a workflow spec and how to create nested workflows.  Be sure to read the comments. They provide useful explanations.
```
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
    - - name: hello1            #hello1 is run before the following steps
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "hello1"
    - - name: hello2a           #double dash => run after previous step
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "hello2a"
      - name: hello2b           #single dash => run in parallel with previous step
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
The above workflow spec prints three different flavors of "hello".
The `hello-hello-hello` template consists of three `steps`.
The first step named `hello1` will be run in sequence whereas the next two steps named `hello2a` and `hello2b` will be run in parallel with each other.
Using the argo cli command, we can graphically display the execution history of this workflow spec, which shows that the steps named `hello2a` and `hello2b` ran in parallel with each other.
```
STEP                                     PODNAME
 ✔ arguments-parameters-rbm92   
 ├---✔ hello1                   steps-rbm92-2023062412
 └-·-✔ hello2a                  steps-rbm92-685171357
   └-✔ hello2b                  steps-rbm92-634838500
```

## Artifacts

**Note:** 
You will need to have installed and configured an artifact repository for this example to run.
[Installing an artifact repository here](https://github.com/argoproj/argo/blob/master/demo.md#5-install-an-artifact-repository).

When running workflows, it is very common to have steps that generate or consume artifacts. Often, the output artifacts of one step may be used as input artifacts to a subsequent step.

The below workflow spec consists of two steps that run in sequence. The first step named `generate-artifact` will generate an artifact using the `whalesay` template which will be consumed by the second step named `print-message` that consumes the generated artifact. 
```
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
The `whalesay` template uses the `cowsay` command to generate a file named `/tmp/hello-world.txt`. It then `outputs` this file as an artifact named `hello-art`. In general, the artifact's `path` may be a directory rather than just a file.
The `print-message` template takes an input artifact named `message`, unpacks it at the `path` named `/tmp/message` and then prints the contents of `/tmp/message` using the `cat` command.
The `artifact-example` template passes the `hello-art` artifact generated as an output of the `generate-artifact` step as the `message` input artifact to the `print-message` step.

## The Structure of Workflow Specs

We now know enough about the basic components of a workflow spec to review its basic structure. 
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

Note that the controller section of the workflow spec will accept the same options as the controller section of a pod spec, including but not limited to env vars, secrets, and volume mounts. Similarly, for volume claims and volumes.

## Secrets
Argo supports the same secrets syntax and mechanisms as Kubernetes Pod specs, which allows access to secrets as environment variables or volume mounts.
- https://kubernetes.io/docs/concepts/configuration/secret/

```
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
Often times, we just want a template that executes a script specified as a here-script (aka. here document) in the workflow spec.
```
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
      image: debian:9.1
      command: [bash]
      source: |                                         # Contents of the here-script
        cat /dev/urandom | od -N2 -An -i | awk -v f=1 -v r=100 '{printf "%i\n", f + r * $1 / 65536}'

  - name: gen-random-int-python
    script:
      image: python:3.6
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
The `script` keyword allows the specification of the script body using the `source` tag. This creates a temporary file containing the script body and then passes the name of the temporary file as the final parameter to `command`, which should be an interpreter that executes the script body.. 

The use of the `script` feature also assigns the standard output of running the script to a special output parameter named `result`. This allows you to use the result of running the script itself in the rest of the workflow spec. In this example, the result is simply echoed by the print-message template.

## Loops

When writing workflows, it is often very useful to be able to iterate over a set of inputs.
```
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
        withItems:              #invoke whalesay once for each item in parallel
        - hello world           #item 1
        - goodbye world         #item 2
            
  - name: whalesay 
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
```

We can also iterate over a sets of items.
```
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

We can pass lists of items as parameters.
```
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: loops-param-arg-
spec:
  entrypoint: loop-param-arg-example
  arguments:
    parameters:
    - name: os-list                                     #a list of items
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
        withParam: "{{inputs.parameters.os-list}}"      #parameter specifies the list to intereate over

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

We can even dynamically generate the list of items to interate over!
```
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
      image: python:3.6
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
We also support conditional execution.
```
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
        template: heads                 #invoke heads template if "heads"
        when: "{{steps.flip-coin.outputs.result}} == heads"
      - name: tails
        template: tails                 #invoke tails template if "tails"
        when: "{{steps.flip-coin.outputs.result}} == tails"
    
  # Return heads or tails based on a random number
  - name: flip-coin
    script:
      image: python:3.6
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
```

## Recursion
Templates can recursively invoke each other! In this variation of the above coin-flip template, we continue to flip coins until it comes up heads.
```
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
        template: heads                 #invoke heads template if "heads"
        when: "{{steps.flip-coin.outputs.result}} == heads"
      - name: tails                     #keep flipping coins if "tails"
        template: coinflip
        when: "{{steps.flip-coin.outputs.result}} == tails"
    
  - name: flip-coin
    script:
      image: python:3.6
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
```
argo get coinflip-recursive-tzcb5

STEP                         PODNAME                              MESSAGE
 ✔ coinflip-recursive-vhph5                                       
 ├---✔ flip-coin             coinflip-recursive-vhph5-2123890397  
 └-·-✔ heads                 coinflip-recursive-vhph5-128690560   
   └-○ tails                                                      

STEP                          PODNAME                              MESSAGE
 ✔ coinflip-recursive-tzcb5                                        
 ├---✔ flip-coin              coinflip-recursive-tzcb5-322836820   
 └-·-○ heads                                                       
   └-✔ tails                                                       
     ├---✔ flip-coin          coinflip-recursive-tzcb5-1863890320  
     └-·-○ heads                                                   
       └-✔ tails                                                   
         ├---✔ flip-coin      coinflip-recursive-tzcb5-1768147140  
         └-·-○ heads                                               
           └-✔ tails                                               
             ├---✔ flip-coin  coinflip-recursive-tzcb5-4080411136  
             └-·-✔ heads      coinflip-recursive-tzcb5-4080323273  
               └-○ tails                                           
```
In the first run, the coin immediately comes up heads and we stop.
In the second run, the coin comes up tail three times before it finally comes up heads and we stop.

## Exit handlers

An exit handler is a template that always executes, irrespective of success or failure, at the end of the workflow.

Some common use cases of exit handlers are:
- cleaning up after a workflow runs
- sending notifications of workflow status (e.g. e-mail/slack)
- posting the pass/fail status to a webhook result (e.g. github build result)
- resubmitting or submitting another workflow

```
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: exit-handlers-
spec:
  entrypoint: intentional-fail
  onExit: exit-handler                  #invoke exit-hander template at end of the workflow
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
To limit the elapsed time for a workflow, you can set `activeDeadlineSeconds`.
```
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
    activeDeadlineSeconds: 10           #terminate container template after 10 seconds
```

## Volumes
The following example dynamically creates a volume and then uses the volume in a two step workflow.
```
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: volumes-pvc-
spec:
  entrypoint: volumes-pvc-example
  volumeClaimTemplates:                 #define volume, same syntax as k8s Pod spec
  - metadata:
      name: workdir                     #name of volume claim
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 1Gi                  #Gi => 1024 * 1024 * 1024

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
      volumeMounts:                     #same syntax as k8s Pod spec
      - name: workdir
        mountPath: /mnt/vol

  - name: print-message
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo getting message from volume; find /mnt/vol; cat /mnt/vol/hello_world.txt"]
      # Mount workdir volume at /mnt/vol before invoking docker/whalesay
      volumeMounts:                     #same syntax as k8s Pod spec
      - name: workdir
        mountPath: /mnt/vol

```
Volumes are a very useful way to move large amounts of data from one step in a workflow to another.
Depending on the system, some volumes may be accessible concurrently from multiple steps.

In some cases, you want to access an already existing volume rather than creating/destroying one dynamically.
```
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

## Daemon Containers
Argo workflows can start containers that run in the background (aka. daemon containers) while the workflow itself continues execution. The daemons will be automatically destroyed when the workflow exits the template scope in which the daemon was invoked. Deamons containers are useful for starting up services to be tested or to be used in testing (aka. fixtures). We also find it very useful when running large simulations to spin up a database as a daemon for collecting and organizing the results. The big advantage of daemons compared with sidecars is that their existance can persist across multiple steps or even the entire workflow.
```
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
        template: influxdb              #start an influxdb as a daemon (see the influxdb template spec below)

    - - name: init-database             #initialize influxdb
        template: influxdb-client
        arguments:
          parameters:
          - name: cmd
            value: curl -XPOST 'http://{{steps.influx.ip}}:8086/query' --data-urlencode "q=CREATE DATABASE mydb"

    - - name: producer-1                #add entries to influxdb
        template: influxdb-client
        arguments:
          parameters:
          - name: cmd
            value: for i in $(seq 1 20); do curl -XPOST 'http://{{steps.influx.ip}}:8086/write?db=mydb' -d "cpu,host=server01,region=uswest load=$i" ; sleep .5 ; done
      - name: producer-2                #add entries to influxdb
        template: influxdb-client
        arguments:
          parameters:
          - name: cmd
            value: for i in $(seq 1 20); do curl -XPOST 'http://{{steps.influx.ip}}:8086/write?db=mydb' -d "cpu,host=server02,region=uswest load=$((RANDOM % 100))" ; sleep .5 ; done
      - name: producer-3                #add entries to influxdb
        template: influxdb-client
        arguments:
          parameters:
          - name: cmd
            value: curl -XPOST 'http://{{steps.influx.ip}}:8086/write?db=mydb' -d 'cpu,host=server03,region=useast load=15.4'

    - - name: consumer                  #consume intries from influxdb
        template: influxdb-client
        arguments:
          parameters:
          - name: cmd
            value: curl --silent -G http://{{steps.influx.ip}}:8086/query?pretty=true --data-urlencode "db=mydb" --data-urlencode "q=SELECT * FROM cpu"

  - name: influxdb
    daemon: true                        #start influxdb as a daemon      
    container:
      image: influxdb:1.2
      restartPolicy: Always             #restart container if it fails
      readinessProbe:                   #wait for readinessProbe to succeed
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

## Sidecars
A sidecar is another container that executes concurrently in the same pod as the "main" container and is useful
in creating multi-container pods.
```
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
In the above example, we create a sidecar container that runs nginx as a simple web server. The order in which containers may come up is random. This is why the 'main' container polls the nginx container until it is ready to service requests. This is a good design pattern when designing multi-container systems. Always wait for any services you need to come up before running your main code.

## Hardwired Artifacts
With Argo, you can use any container image that you like to generate any kind of artifact. In practice, however, we find certain types of artifacts are very common and provide a more convenient way to generate and use these artifacts. In particular, we have "hardwired" support for git, http and s3 artifacts.
```
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
          repo: https://github.com/argoproj/argo.git
          revision: "master"
      # Download kubectl 1.8.0 and place it at /bin/kubectl
      - name: kubectl
        path: /bin/kubectl
        mode: 755
        http:
          url: https://storage.googleapis.com/kubernetes-release/release/v1.8.0/bin/linux/amd64/kubectl
      # Copy an s3 bucket and place it at /s3
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

In many cases, you will want to manage Kuberenetes resources from Argo workflows. The resource template allows you to create, delete or updated any type of kubernetes resource.
```
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
    resource:                   #indicates that this is a resource template
      action: create            #can be any kubectl action (e.g. create, delete, apply, patch)
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

## Docker-in-Docker (aka. DinD) Using Sidecars
An application of sidecars is to implement DinD (Docker-in-Docker).
DinD is useful when you want to run Docker commands from inside a container. For example, you may want to build and push a container image from inside your build container. In the following example, we use the docker:dind container to run a Docker daemon in a sidecar and give the main container access to the daemon.
```
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: sidecar-dind-
spec:
  entrypoint: dind-sidecar-example
  templates:
  - name: dind-sidecar-example
    container:
      image: docker:17.10
      command: [sh, -c]
      args: ["until docker ps; do sleep 3; done; docker run --rm debian:latest cat /etc/os-release"]
      env:
      - name: DOCKER_HOST               #the docker daemon can be access on the standard port on localhost
        value: 127.0.0.1
    sidecars:
    - name: dind
      image: docker:17.10-dind          #Docker already provides an image for running a Docker daemon
      securityContext:
        privileged: true                #the Docker daemon can only run in a privileged container
      # mirrorVolumeMounts will mount the same volumes specified in the main container
      # to the sidecar (including artifacts), at the same mountPaths. This enables
      # dind daemon to (partially) see the same filesystem as the main container in
      # order to use features such as docker volume binding.
      mirrorVolumeMounts: true
```

## Continuous integration example
Continuous integration is a popular appication for workflows. Currently, Argo does not provide event triggers for automatically kicking off your CI jobs, but we plan to do so in the near future. Until then, you can easily write a cron job that checks for new commits and kicks off the needed workflow, or use your existing Jenkins server to kick off the workflow.

A good example of a CI workflow spec is provided at https://github.com/argoproj/argo/tree/master/examples/influxdb-ci.yaml. Because it just uses the concepts that we've already covered and is somewhat long, we don't go into details here.

