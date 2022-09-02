# Hello World

Let's start by creating a very simple workflow template to echo "hello world" using the `docker/whalesay` container
image from Docker Hub.

You can run this directly from your shell with a simple docker command:

```bash
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

Below, we run the same container on a Kubernetes cluster using an Argo workflow template. Be sure to read the comments
as they provide useful explanations.

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
        command: [ cowsay ]
        args: [ "hello world" ]
        resources: # limit the resources
          limits:
            memory: 32Mi
            cpu: 100m
```

Argo adds a new `kind` of Kubernetes spec called a `Workflow`. The above spec contains a single `template`
called `whalesay` which runs the `docker/whalesay` container and invokes `cowsay "hello world"`. The `whalesay` template
is the `entrypoint` for the spec. The entrypoint specifies the initial template that should be invoked when the workflow
spec is executed by Kubernetes. Being able to specify the entrypoint is more useful when there is more than one template
defined in the Kubernetes workflow spec. :-)
