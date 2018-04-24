# Argo - The Workflow Engine for Kubernetes

![Argo Image](argo.png)

## News

We are excited to welcome Cyrus Biotechnolgy, Google and NVIDIA as corporate members of the Argo Community! They have been active users of Argo for some time now and have decided to increase their participation in both the use and development of Argo. If you actively use Argo at your company and believe that your company may be interested in actively participating in the Argo Community, please ask a representative to contact saradhi_sreegiriraju@intuit.com for additional information.

Updated community documents, including CLAs, are available [here](https://github.com/argoproj/argo/tree/master/community). These documents are essentially identical to those used by CNCF projects such as Kubernetes and are designed to protect Argo's contributors, users and Intuit, the current custodian of the Argo Project.

We will be scheduling the first Argo Community Meeting for the beginning of May to get to know each other, review the current Argo projects, and gather feedback for future projects. Please stay tuned for additional details.

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

## Argo 2.0
Argo 2.0 is a Kubernetes Custom Resource Definition (CRD) which can run workflows using kubectl commands.

* [Get started here](https://github.com/argoproj/argo/blob/master/demo.md)
* [How to write Argo workflow specs](https://github.com/argoproj/argo/blob/master/examples/README.md)
* [How to configure your artifact repository](https://github.com/argoproj/argo/blob/master/ARTIFACT_REPO.md)

## Who uses Argo?
As the Argo Community grows, we'd like to keep track of our users. Please send a PR with your company name and @githubhandle if you may.

Currently **officially** using Argo:

1. Cyrus Biotechnology
1. Google
1. Intuit [[@mukulikak](https://github.com/mukulikak)]
1. NVIDIA

## Presentations
* TGI Kubernetes with Joe Beda: [Argo workflow system](https://www.youtube.com/watch?v=M_rxPPLG8pU&start=859)

## Community Blogs
* [Argo integration review](http://dev.matt.hillsdon.net/2018/03/24/argo-integration-review.html)
* Please share your own thoughs

## Project Resources
* Argo GitHub:  https://github.com/argoproj
* Argo Slack:   [click here to join](https://join.slack.com/t/argoproj/shared_invite/enQtMzExODU3MzIyNjYzLTA5MTFjNjI0Nzg3NzNiMDZiNmRiODM4Y2M1NWQxOGYzMzZkNTc1YWVkYTZkNzdlNmYyZjMxNWI3NjY2MDc1MzI)
* Argo website: https://argoproj.github.io/argo-site
* Argo forum:   https://groups.google.com/forum/#!forum/argoproj
