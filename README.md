# Argo - The Workflow Engine for Kubernetes

![Argo Image](argo.png)

## What is Argo?
Argo is an open source container-native workflow engine for developing and running applications on Kubernetes.
* Define workflows where each step in the workflow is a container.
* Run rich CI/CD workflows using Docker-in-Docker, complex testing with built in artifact management, secret management and lifecycle management of dev/test resources.
* Build, test and deploy scalable stateful and stateless cloud-native apps and microservices.

## Why Argo?
* Argo is designed from the ground up for containers without the baggage and limitations of legacy VM and server-based environments.
* Argo is cloud agnostic. Today we support AWS and GKE (alpha) with additional platforms coming soon.
* With Argo, you don’t need to install or learn other tools such as Jenkins, Chef, Cloud Formation... 

## Getting started

### Step 1: Download the argo binary

**Mac:** `curl -sSL -O https://s3-us-west-1.amazonaws.com/ax-public/argocli/latest/darwin_amd64/argo`

**Linux:** `curl -sSL -O https://s3-us-west-1.amazonaws.com/ax-public/argocli/latest/linux_amd64/argo`

```
chmod a+x ./argo
cp ./argo /usr/local/bin
```

### Step 2: Install argo

`argo cluster`

## Main Features
* Container-native workflows for Kubernetes.
  * Each step in the workflow is a container
  * Arbitrarily compose sub-workflows to create larger workflows
  * No need to install or learn other tools such as Jenkins, Chef, Cloud Formation
* Configuration as code (YAML for everything)
* Built-in support for artifacts, persistent volumes, and DNS/load-balancers/firewalls.
* DinD (Docker-in-Docker) out of the box. Run docker builds and other containers from within containerized workflows.
* "Cashboard" shows cost of running a workflow. Also, spending per user and application.
* Managed fixtures.

## Resources
* Argo website: https://argoproj.github.io/argo-site
* Argo GitHub:  https://github.com/argoproj
* Argo forum:   https://groups.google.com/forum/#!forum/argoproj

