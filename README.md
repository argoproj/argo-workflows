[![slack](https://img.shields.io/badge/slack-argoproj-brightgreen.svg?logo=slack)](https://argoproj.github.io/community/join-slack)

# Argoproj - Get stuff done with Kubernetes

![Argo Image](argo.png)

## Quickstart
```bash
kubectl create namespace argo
kubectl apply -n argo -f https://raw.githubusercontent.com/argoproj/argo/stable/manifests/install.yaml
```

## News

KubeCon 2018 in Seattle was the biggest KubeCon yet with 8000 developers attending. We connected with many existing and new Argoproj users and contributions, and gave away a lot of Argo T-shirts at our booth sponsored by Intuit!

We were also super excited to see KubeCon presentations about Argo by Argo developers, users and partners.
* [CI/CD in Light Speed with K8s and Argo CD](https://www.youtube.com/watch?v=OdzH82VpMwI&feature=youtu.be)
  * How Intuit uses Argo CD.
* [Automating Research Workflows at BlackRock](https://www.youtube.com/watch?v=ZK510prml8o&t=0s&index=169&list=PLj6h78yzYM2PZf9eA7bhWnIh_mK1vyOfU)
  * Why BlackRock created Argo Events and how they use it.
* [Machine Learning as Code](https://www.youtube.com/watch?v=VXrGp5er1ZE&t=0s&index=135&list=PLj6h78yzYM2PZf9eA7bhWnIh_mK1vyOfU)
  * How Kubeflow uses Argo Workflows as its core workflow engine and Argo CD to declaratively deploy ML pipelines and models.

If you actively use Argo in your organization and your organization would be interested in participating in the Argo Community, please ask a representative to contact saradhi_sreegiriraju@intuit.com for additional information.

## What is Argoproj?

Argoproj is a collection of tools for getting work done with Kubernetes.
* [Argo Workflows](https://github.com/argoproj/argo) - Container-native Workflow Engine
* [Argo CD](https://github.com/argoproj/argo-cd) - Declarative GitOps Continuous Delivery
* [Argo Events](https://github.com/argoproj/argo-events) - Event-based Dependency Manager
* [Argo Rollouts](https://github.com/argoproj/argo-rollouts) - Deployment CR with support for Canary and Blue Green deployment strategies

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

## Documentation
* [Get started here](demo.md)
* [How to write Argo Workflow specs](examples/README.md)
* [How to configure your artifact repository](ARTIFACT_REPO.md)

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

## Who uses Argo?
As the Argo Community grows, we'd like to keep track of our users. Please send a PR with your organization name.

Currently **officially** using Argo:

1. [Adevinta](https://www.adevinta.com/)
1. [Admiralty](https://admiralty.io/)
1. [Adobe](https://www.adobe.com/)
1. [Alibaba Cloud](https://www.alibabacloud.com/about)
1. [BlackRock](https://www.blackrock.com/)
1. [Canva](https://www.canva.com/)
1. [Codec](https://www.codec.ai/)
1. [CoreFiling](https://www.corefiling.com/)
1. [Cratejoy](https://www.cratejoy.com/)
1. [Cyrus Biotechnology](https://cyrusbio.com/)
1. [Datadog](https://www.datadoghq.com/)
1. [DataStax](https://www.datastax.com/)
1. [Equinor](https://www.equinor.com/)
1. [Gardener](https://gardener.cloud/)
1. [Gladly](https://gladly.com/)
1. [GitHub](https://github.com/)
1. [Google](https://www.google.com/intl/en/about/our-company/)
1. [IBM](https://ibm.com)
1. [Interline Technologies](https://www.interline.io/blog/scaling-openstreetmap-data-workflows/)
1. [Intuit](https://www.intuit.com/)
1. [Karius](https://www.kariusdx.com/)
1. [KintoHub](https://www.kintohub.com/)
1. [Localytics](https://www.localytics.com/)
1. [Max Kelsen](https://maxkelsen.com/)
1. [Mirantis](https://mirantis.com/)
1. [NVIDIA](https://www.nvidia.com/)
1. [OVH](https://www.ovh.com/)
1. [Preferred Networks](https://www.preferred-networks.jp/en/)
1. [Quantibio](http://quantibio.com/us/en/)
1. [SAP Fieldglass](https://www.fieldglass.com/)
1. [SAP Hybris](https://cx.sap.com/)
1. [Styra](https://www.styra.com/)
1. [Threekit](https://www.threekit.com/)
1. [Commodus Tech](https://www.commodus.tech)


## Community Blogs and Presentations
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
* Argo Slack:   [click here to join](https://join.slack.com/t/argoproj/shared_invite/enQtMzExODU3MzIyNjYzLWUxZDYyODIyYzY3N2RjOWMyNDA4NmFjMTNkMTE1ODI2OGY3MzgyMWFmMmY3N2UzNWRmOWFmMGY4NTBhZGQxYWY)
