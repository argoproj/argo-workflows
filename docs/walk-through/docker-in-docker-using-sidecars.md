# Docker-in-Docker Using Sidecars

Note: It is increasingly unlikely that the below example will work for you on your version of Kubernetes. [Since Kubernetes 1.24, the dockershim has been unavailable as part of Kubernetes](https://kubernetes.io/blog/2022/02/17/dockershim-faq/), rendering Docker-in-Docker unworkable. It is recommended to seek alternative methods of building containers, such as [Kaniko](https://github.com/GoogleContainerTools/kaniko) or [Buildkit](https://github.com/moby/buildkit). A [Buildkit Workflow example](https://raw.githubusercontent.com/argoproj/argo-workflows/main/examples/buildkit-template.yaml) is available in the examples directory of the Argo Workflows repository.

---

An application of sidecars is to implement Docker-in-Docker (DIND). DIND is useful when you want to run Docker commands from inside a container. For example, you may want to build and push a container image from inside your build container. In the following example, we use the `docker:dind` image to run a Docker daemon in a sidecar and give the main container access to the daemon.

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
