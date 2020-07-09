# Windows Container Support

The Argo server and the workflow controller currently only run on Linux. The workflow executor however also runs on Windows nodes, meaning you can use Windows containers inside your workflows! Here are the steps to get started.

## Requirements
* Kubernetes 1.14 or later, supporting Windows nodes
* Hybrid cluster containing Linux and Windows nodes like described in the [Kubernetes docs](https://kubernetes.io/docs/setup/production-environment/windows/user-guide-windows-containers/)
* Argo configured and running like described [here](quick-start.md) 

## Schedule workflows with Windows containers

If you're running workflows in your hybrid Kubernetes cluster, always make sure to include a `nodeSelector` to run the steps on the correct host OS:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-windows-
spec:
  entrypoint: hello-win
  templates:
    - name: hello-win
      nodeSelector:
        kubernetes.io/os: windows    # specify the OS your step should run on
      container:
        image: mcr.microsoft.com/windows/nanoserver:1809
        command: ["cmd", "/c"]
        args: ["echo", "Hello from Windows Container!"]
```

You can run this example and get the logs:
```
$ argo submit --watch https://raw.githubusercontent.com/argoproj/argo/master/examples/hello-windows.yaml
$ argo logs hello-windows-s9kk5
hello-windows-s9kk5: "Hello from Windows Container!"
```

## Schedule hybrid workflows

You can also run different steps on different host OSs. This can for example be very helpful when you need to compile your application on Windows and Linux.

An example workflow can look like the following:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-hybrid-
spec:
  entrypoint: mytemplate
  templates:
    - name: mytemplate
      steps:
        - - name: step1
            template: hello-win
        - - name: step2
            template: hello-linux

    - name: hello-win
      nodeSelector:
        kubernetes.io/os: windows
      container:
        image: mcr.microsoft.com/windows/nanoserver:1809
        command: ["cmd", "/c"]
        args: ["echo", "Hello from Windows Container!"]
    - name: hello-linux
      nodeSelector:
        kubernetes.io/os: linux
      container:
        image: alpine
        command: [echo]
        args: ["Hello from Linux Container!"]

```

Again, you can run this example and get the logs:
```
$ argo submit --watch https://raw.githubusercontent.com/argoproj/argo/master/examples/hello-hybrid.yaml
$ argo logs hello-hybrid-plqpp
hello-hybrid-plqpp-1977432187: "Hello from Windows Container!"
hello-hybrid-plqpp-764774907: Hello from Linux Container!
```

## Building the workflow executor image for Windows

To build the workflow executor image for Windows you need a Windows machine running Windows Server 2019 with Docker installed like described [in the docs](https://docs.docker.com/ee/docker-ee/windows/docker-ee/#install-docker-engine---enterprise).

You then clone the project and run the Docker build with the Dockerfile for Windows and `argoexec` as a target:

```
git clone https://github.com/argoproj/argo.git
cd argo
docker build -t myargoexec -f .\Dockerfile.windows --target argoexec .
```