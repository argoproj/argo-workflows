# Argo - The Workflow Engine for Kubernetes

![Argo Image](argo.png)

## What is Argo?
Argo is an open source container-native workflow engine for getting work done on Kubernetes. Argo is implemented as a Kubernetes CRD (Customer Resource Definition).

* Define workflows where each step in the workflow is a container.
  * Soon, you will also be able to define workflows as a dependency graph (DAG).
* Easily leverage Kubernetes to run compute intensive jobs like data processing or machine learning jobs in a fraction of the time using Argo workflows.
* Run rich CI/CD pipelines using Docker-in-Docker, built-in artifact management, secret management and native access to other Kubernetes resources.


## Why Argo?
* Argo is designed from the ground up for containers without the overhead and limitations of legacy VM and server-based environments.
* Argo is cloud agnostic and can run on any kubernetes cluster.
* Argo with Kubernetes puts a cloud-scale supercomputer at your fingertips.
* We want to make it as easy to run Aggo workflows on Kubernetes as it is to run jobs on you laptop.

## Argo 2.0 Alpha 
Argo 2.0 is a Kubernetes Custom Resource Definition (CRD) which can run workflows using kubectl commands.

* [Get started here](https://github.com/argoproj/argo/blob/master/demo.md)
* [How to write Argo workflow specs](https://github.com/argoproj/argo/blob/master/examples/README.md)

## Resources
* Argo website: https://argoproj.github.io/argo-site
* Argo GitHub:  https://github.com/argoproj
* Argo Slack:   [click here to join](https://join.slack.com/t/argoproj/shared_invite/enQtMjkyNjcxMDg5NTM2LWFiMDJlZWVhYWI2NmI3OWQyNTZjZThjN2UwNGFlYTJkNGM5ODg0MGJkZTFjMGRhZjQ1MzAzNWY1NzlhZjI2MDg)
* Argo forum:   https://groups.google.com/forum/#!forum/argoproj
