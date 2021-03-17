<p align="center"><img src="./assets/argo.png" alt="Argo Logo"></p>

[![CI](https://github.com/argoproj/argo-workflows/workflows/CI/badge.svg)](https://github.com/argoproj/argo-workflows/actions?query=event%3Apush+branch%3Amaster)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/3830/badge)](https://bestpractices.coreinfrastructure.org/projects/3830)
[![slack](https://img.shields.io/badge/slack-argoproj-brightgreen.svg?logo=slack)](https://argoproj.github.io/community/join-slack)

## What is Argo Workflows?
Argo Workflows is an open source container-native workflow engine for orchestrating parallel jobs on Kubernetes. Argo Workflows is implemented as a Kubernetes CRD (Custom Resource Definition).

* Define workflows where each step in the workflow is a container.
* Model multi-step workflows as a sequence of tasks or capture the dependencies between tasks using a directed acyclic graph (DAG).
* Easily run compute intensive jobs for machine learning or data processing in a fraction of the time using Argo Workflows on Kubernetes.
* Run CI/CD pipelines natively on Kubernetes without configuring complex software development products.

Argo is a [Cloud Native Computing Foundation (CNCF)](https://cncf.io/) hosted project.

[![Argo Workflows in 5 minutes](https://img.youtube.com/vi/TZgLkCFQ2tk/0.jpg)](https://www.youtube.com/watch?v=TZgLkCFQ2tk)

## Why Argo Workflows?
* Designed from the ground up for containers without the overhead and limitations of legacy VM and server-based environments.
* Cloud agnostic and can run on any Kubernetes cluster.
* Easily orchestrate highly parallel jobs on Kubernetes.
* Argo Workflows puts a cloud-scale supercomputer at your fingertips!

# Argo Documentation

### Getting Started

For set-up information and running your first Workflows, please see our [Getting Started](quick-start.md) guide.

### Examples

For detailed examples about what Argo can do, please see our [documentation by example](https://argoproj.github.io/argo-workflows/examples/) page.

### Fields

For a full list of all the fields available in for use in Argo, and a link to examples where each is used, please see [Argo Fields](fields.md).
