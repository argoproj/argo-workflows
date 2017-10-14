# Argo Lite

Argo Lite is a lightweight workflow engine that executes container-native workflows defined using [Argo YAML Domain-Specific Language (DSL)](https://argoproj.github.io/docs/yaml/dsl_reference_intro.html).  Argo Lite implements the same APIs as [Argo](https://github.com/argoproj/argo). This allows you to execute Argo Lite with both [Argo CLI](https://argoproj.github.io/docs/dev-cli-reference.html) and Argo UI. Currently, Argo Lite supports Docker and Kubernetes as the backend container execution engines.

## Argo Lite will be released in mid-October

Argo Lite is not yet fully tested and may crash under load. Early testing/contributions are very welcome.

## Why?

Argo Lite may be used to quickly experience [Argo](https://github.com/argoproj/argo) workflows without deploying a complete Kubernetes cluster or to debug Argo workflows locally on your laptop.

## Try it

Prerequisite: The [Argo CLI](https://applatix.com/open-source/argo/get-started/installation) must be installed first before you start using Argo Lite.

### On your laptop:

* *Using Docker*

 1. Run Argo Lite server:

   ```
   docker run --rm -p 8080:8080  -v /var/run/docker.sock:/var/run/docker.sock -dt argoproj/argo-lite node /app/dist/main.js -u /app/dist/ui

   ```

 2. Configure [Argo CLI](https://argoproj.github.io/docs/dev-cli-reference.html) to talk to your Argo Lite instance:

    ```

    argo login --config argo-lite http://localhost:8080 --username test --password test

    ```

* *Using Minikube*

 NOTE: Before you use Minikube, you must have installed a hypervisor, `kubectl` (command-line for a Kubernetes cluster), and minikube. For instructions, see [Install Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/).

 1. Create Argo Lite deployment

   *Manually*

   ```

   # Argo Lite UI is available at http://localhost:8080
   curl -o /tmp/argo.yaml https://raw.githubusercontent.com/argoproj/argo/master/lite/argo-lite.yaml && kubectl create -f /tmp/argo.yaml

   ```

   *Using [helm](https://docs.helm.sh/using_helm/#installing-helm):*

   ```
   helm repo add argo https://argoproj.github.io/argo-helm
   kubectl config view

   ```

 2. Configure [Argo CLI](https://argoproj.github.io/docs/dev-cli-reference.html) to talk to your Argo Lite instance:

   ```

   # Argo Lite UI is available at http://<deployed Argo Lite service URL>
   argo login --config argo-lite-kube <deployed Argo Lite service URL> --username test --password test

   ```

### On your Kubernetes cluster:

1. Create Argo Lite deployment

  *Manually*

  ```

  # Argo Lite UI is available at http://localhost:8080
  curl -o /tmp/argo.yaml https://raw.githubusercontent.com/argoproj/argo/master/lite/argo-lite.yaml && kubectl create -f /tmp/argo.yaml

  ```

  *Using [helm](https://docs.helm.sh/using_helm/#installing-helm):*

  ```
  helm repo add argo https://argoproj.github.io/argo-helm
  kubectl config view

  ```

2. Configure [Argo CLI](https://argoproj.github.io/docs/dev-cli-reference.html) to talk to your Argo Lite instance:

  ```

  # Argo Lite UI is available at http://<deployed Argo Lite service URL>
  argo login --config argo-lite-kube <deployed Argo Lite service URL> --username test --password test

  ```

### Run the Sample Workflows

Build Argo Lite using Argo Lite :-) The YAML template file **Argo Lite CI** is defined in the [.argo folder](https://github.com/argoproj/argo/blob/master/.argo/lite-ci.yaml).

```
git clone https://github.com/argoproj/argo.git && cd argo && argo job submit 'Argo Lite CI' --config argo-lite --local
```

![alt text](./demo.gif "Logo Title Text 1"
