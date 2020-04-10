[![slack](https://img.shields.io/badge/slack-argoproj-brightgreen.svg?logo=slack)](https://argoproj.github.io/community/join-slack)
[![CircleCI](https://circleci.com/gh/argoproj/argo.svg?style=svg)](https://circleci.com/gh/argoproj/argo)
[![codecov](https://codecov.io/gh/argoproj/argo/branch/master/graph/badge.svg)](https://codecov.io/gh/argoproj/argo)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=argoproj_argo&metric=alert_status)](https://sonarcloud.io/dashboard?id=argoproj_argo)

# Argoproj - Get stuff done with Kubernetes

![Argo Image](docs/assets/argo.png)

## Quickstart
```bash
kubectl create namespace argo
kubectl apply -n argo -f https://raw.githubusercontent.com/argoproj/argo/stable/manifests/install.yaml
```

## What is Argoproj?

Argoproj is a collection of tools for getting work done with Kubernetes.
* [Argo Workflows](https://github.com/argoproj/argo) - Container-native Workflow Engine
* [Argo CD](https://github.com/argoproj/argo-cd) - Declarative GitOps Continuous Delivery
* [Argo Events](https://github.com/argoproj/argo-events) - Event-based Dependency Manager
* [Argo Rollouts](https://github.com/argoproj/argo-rollouts) - Progressive Delivery with support for Canary and Blue Green deployment strategies

Also argoproj-labs is a separate GitHub org that we setup for community contributions related to the Argoproj ecosystem. Repos in argoproj-labs are administered by the owners of each project. Please reach out to us on the Argo slack channel if you have a project that you would like to add to the org to make it easier to others in the Argo community to find, use, and contribute back.
* https://github.com/argoproj-labs

## What is Argo Workflows?
Argo Workflows is an open source container-native workflow engine for orchestrating parallel jobs on Kubernetes. Argo Workflows is implemented as a Kubernetes CRD (Custom Resource Definition).

* Define workflows where each step in the workflow is a container.
* Model multi-step workflows as a sequence of tasks or capture the dependencies between tasks using a graph (DAG).
* Easily run compute intensive jobs for machine learning or data processing in a fraction of the time using Argo Workflows on Kubernetes.
* Run CI/CD pipelines natively on Kubernetes without configuring complex software development products.

## Why Argo Workflows?
* Designed from the ground up for containers without the overhead and limitations of legacy VM and server-based environments.
* Cloud agnostic and can run on any Kubernetes cluster.
* Easily orchestrate highly parallel jobs on Kubernetes.
* Argo Workflows puts a cloud-scale supercomputer at your fingertips!

## Who uses Argo Workflows?
[Official Argo Workflows user list](USERS.md)

## Documentation
* [Get started here](docs/getting-started.md)
* [How to write Argo Workflow specs](examples/README.md)
* [How to configure your artifact repository](docs/configure-artifact-repository.md)

## Features
* DAG or Steps based declaration of workflows
* Artifact support (S3, Artifactory, HTTP, Git, raw)
* Step level input & outputs (artifacts/parameters)
* Loops
* Parameterization
* Conditionals
* Timeouts (step & workflow level)
* Retry (step & workflow level)
* Resubmit (memoized)
* Suspend & Resume
* Cancellation
* K8s resource orchestration
* Exit Hooks (notifications, cleanup)
* Garbage collection of completed workflow
* Scheduling (affinity/tolerations/node selectors)
* Volumes (ephemeral/existing)
* Parallelism limits
* Daemoned steps
* DinD (docker-in-docker)
* Script steps

## Community Blogs and Presentations
* [Argo Ansible role: Provisioning Argo Workflows on OpenShift](https://medium.com/@marekermk/provisioning-argo-on-openshift-with-ansible-and-kustomize-340a1fda8b50)
* [Argo Workflows vs Apache Airflow](http://bit.ly/30YNIvT)
* [CI/CD with Argo on Kubernetes](https://medium.com/@bouwe.ceunen/ci-cd-with-argo-on-kubernetes-28c1a99616a9)
* [Running Argo Workflows Across Multiple Kubernetes Clusters](https://admiralty.io/blog/running-argo-workflows-across-multiple-kubernetes-clusters/)
* [Open Source Model Management Roundup: Polyaxon, Argo, and Seldon](https://www.anaconda.com/blog/developer-blog/open-source-model-management-roundup-polyaxon-argo-and-seldon/)
* [Producing 200 OpenStreetMap extracts in 35 minutes using a scalable data workflow](https://www.interline.io/blog/scaling-openstreetmap-data-workflows/)
* [Argo integration review](http://dev.matt.hillsdon.net/2018/03/24/argo-integration-review.html)
* TGI Kubernetes with Joe Beda: [Argo workflow system](https://www.youtube.com/watch?v=M_rxPPLG8pU&start=859)
* [Community meeting minutes and recordings](https://docs.google.com/document/d/16aWGQ1Te5IRptFuAIFtg3rONRQqHC1Z3X9rdDHYhYfE)

## Project Resources
* Argo GitHub:  https://github.com/argoproj
* Argo website: https://argoproj.github.io/
* Argo Slack:   [click here to join](https://argoproj.github.io/community/join-slack)
