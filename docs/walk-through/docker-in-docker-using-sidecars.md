# Docker-in-Docker Using Sidecars

!!! Note "Alternatives"
    Alternative methods of building containers, such as [Kaniko](https://github.com/GoogleContainerTools/kaniko) or [Buildkit](https://github.com/moby/buildkit) can be simpler and more secure.
    See the [Buildkit template](https://github.com/argoproj/argo-workflows/blob/main/examples/buildkit-template.yaml) as an example.

You can use [sidecars](sidecars.md) to implement Docker-in-Docker (DIND).
You can use DIND to run Docker commands inside a container, such as to build and push a container image.

In the following example, use the `docker:dind` image to run a Docker daemon in a sidecar and give the main container access to the daemon:

/// tab | YAML

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
      command: [dockerd-entrypoint.sh]
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

///

/// tab | Python

```python
from hera.workflows import Container, UserContainer, Workflow
from hera.workflows.models import EnvVar, SecurityContext

with Workflow(
    generate_name="sidecar-dind-",
    entrypoint="dind-sidecar-example",
) as w:
    Container(
        name="dind-sidecar-example",
        image="docker:19.03.13",
        command=["sh", "-c"],
        args=[
            "until docker ps; do sleep 3; done; docker run --rm debian:latest cat /etc/os-release"
        ],
        sidecars=[
            UserContainer(
                name="dind",
                image="docker:19.03.13-dind",
                command=["dockerd-entrypoint.sh"],
                env=[EnvVar(name="DOCKER_TLS_CERTDIR", value="")],
                mirror_volume_mounts=True,
                security_context=SecurityContext(privileged=True),
            )
        ],
        env=[EnvVar(name="DOCKER_HOST", value="127.0.0.1")],
    )
```

///
