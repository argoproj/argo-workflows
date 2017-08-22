# Argo - The Workflow Engine for Kubernetes

![Argo Image](argo.png)

## What is Argo?
Argo is an open source container-native workflow engine for developing and running applications on Kubernetes.
* Define workflows where each step in the workflow is a container.
* Run rich CI/CD workflows using Docker-in-Docker, Kubernetes-in-Kubernetes, complex testing with built in artifact management, secret management and lifecycle management of dev/test resources.
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


## History
VMs were a big improvement over physical servers when it came to dynamically provisioning and running applications. We found however, that VMs are too clunky and heavyweight for complex automation tasks and orchestrating distributed applications. When we started using containers for the first time, we were very excited by the potential of this lightweight and portable virtualization technology. We started creating a container-native platform for developing and running applications. We started with Mesos because that was the most stable platform for running distributed applicaitons at the time. A year later, we noticed that the Kubernetes community was growing faster and making more progress than any other container orchestration platform. We quickly switched to Kubernetes and haven't look back since :-)

As we worked more with Kubernetes, we discovered that it was lacking an integrated workflow engine for orchestrating jobs as well as deploying complex distributed applications. One typically had to integrate external workflow engines or scripting using kubectl. The result was cumbersome and not portable. This motivated us to create a container-native workflow engine for Kubernetes. We found along the way, that just a core workflow engine is not enough. You also need other services such as artifacts, load balancers etc. to enable the workflows to do useful work. We want to make Kubernetes THE platform for developing and deploying distributed applications.

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
* Argo website: https://argoproj.io
* Argo GitHub:  https://github.com/argoproj
* Argo forum:   https://groups.google.com/forum/#!forum/argoproj

