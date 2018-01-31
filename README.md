# Argo - The Workflow Engine for Kubernetes

![Argo Image](argo.png)

## News
The core Argo team is joining Intuit! We are super excited to take Argo to the next level as a part of Intuit. We remain firmly committed to Open Source, Kubernetes, and our AWESOME users!

Blog post [here](https://blog.argoproj.io/applatix-joins-intuit-7ab587270573).

## What is Argo?
Argo is an open source container-native workflow engine for getting work done on Kubernetes. Argo is implemented as a Kubernetes CRD (Custom Resource Definition).

* Define workflows where each step in the workflow is a container.
  * Soon, you will also be able to define workflows as a dependency graph (DAG).
* Easily leverage Kubernetes to run compute intensive jobs like data processing or machine learning jobs in a fraction of the time using Argo workflows.
* Run rich CI/CD pipelines using Docker-in-Docker, built-in artifact management, secret management and native access to other Kubernetes resources.


## Why Argo?
* Argo is designed from the ground up for containers without the overhead and limitations of legacy VM and server-based environments.
* Argo is cloud agnostic and can run on any kubernetes cluster.
* Argo with Kubernetes puts a cloud-scale supercomputer at your fingertips.
* We want to make it as easy to run Argo workflows on Kubernetes as it is to run jobs on you laptop.

## Argo 2.0 Alpha 
Argo 2.0 is a Kubernetes Custom Resource Definition (CRD) which can run workflows using kubectl commands.

* [Get started here](https://github.com/argoproj/argo/blob/master/demo.md)
* [How to write Argo workflow specs](https://github.com/argoproj/argo/blob/master/examples/README.md)
* [How to configure your artifact repository](https://github.com/argoproj/argo/blob/master/ARTIFACT_REPO.md)

## Presentations
* TGI Kubernetes with Joe Beda: [Argo workflow system](https://www.youtube.com/watch?v=M_rxPPLG8pU&start=859)

## Resources
* Argo GitHub:  https://github.com/argoproj
* Argo Slack:   [click here to join](https://join.slack.com/t/argoproj/shared_invite/enQtMjkyNjcxMDg5NTM2LWFiMDJlZWVhYWI2NmI3OWQyNTZjZThjN2UwNGFlYTJkNGM5ODg0MGJkZTFjMGRhZjQ1MzAzNWY1NzlhZjI2MDg)
* Argo website: https://argoproj.github.io/argo-site
* Argo forum:   https://groups.google.com/forum/#!forum/argoproj
