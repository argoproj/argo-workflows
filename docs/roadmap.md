# Roadmap
## SDKs

Build on the [Java SDK](https://github.com/argoproj-labs/argo-client-java) by stabilising the code base. Continue to improve the stability of SDKs, for example correcting bugs in OpenAPI specifications.

## More Ways to Trigger Workflows

Offer new ways to trigger workflows on top of existing mechanisms such as via the GUI or CLI. 

For example, how can [one workflow trigger another workflow](https://github.com/argoproj/argo-workflows/issues/3295)? Build out-of-the-box support for triggering workflows via [webhooks](https://github.com/argoproj/argo-workflows/issues/2667), so it is easier to integrate with third-party tools such as Github or Gitlab. 

## Controller Enhancements

Provide new features and enhancements that build on existing controller and executor functionality. 

### Memoization

Memoization of workflow steps allows workflow to execute quicker by work-avoidance. This will be improved by adding support for [automatic TTL](https://github.com/argoproj/argo-workflows/issues/3593) and [alternative storage options](https://github.com/argoproj/argo-workflows/issues/3587) to config maps.

### Semaphores and Mutexes

Enable more ways to lock workflows when waiting on resources with the existing semaphore capability and [mutexes](https://github.com/argoproj/argo-workflows/issues/2677).

### Task Level Priorities

Introduce task level priorities to enable fine-tuning of the order of nodes within an workflow execution graph.

### Artifact Management Enhancements

Improve the handling of artifacts loaded from AWS, GCS, Artifactory et al.  For example, [features to support `artifactRepositoryRef`](https://github.com/argoproj/argo-workflows/issues/3307), or [automatically creating buckets](https://github.com/argoproj/argo-workflows/issues/3586).

## Metrics & Reporting

Make it easier to understand how long and how much resource workflows use by supporting [automatic duration prediction](https://github.com/argoproj/argo-workflows/issues/2717) for newly started workflows and [historical workflow reports](https://github.com/argoproj/argo-workflows/issues/3557).

## Scaling, Reliability, Performance

Continue to improve support for large numbers (10,000+) of massive (2,000+ node) workflows.

Be able to show larger workflows in GUI by enabling the filtering and [collapsing of workflow graphs](https://github.com/argoproj/argo-workflows/issues/3527) in the GUI. Make improvements to listing [large numbers of workflows](https://github.com/argoproj/argo-workflows/issues/3590) on GUI by using filtering, sorting, and  pagination; or [grouping workflows](https://github.com/argoproj/argo-workflows/issues/3591) is list view.

Execute [Failure mode effect analysis (FMEA) and stress testing](https://github.com/argoproj/argo-workflows/issues/3751) to identify areas for improvement and then implement changes to make workflows and archiving run more reliably.

## Argo Events Integration

Improve integration with Argo Events to enable users to trigger workflows in more ways and provide clearer linkage between the cause and outcome.

## Workflow Template Catalog

Support and help build out [the workflow template catalog](https://argoproj-labs.github.io/argo-workflows-catalog/), a library of reusable templates for common workflow tasks.

## Use Case Specific Enhancements

We're looking for partners and contributors to help design and develop changes to specifically help the following use cases:

* MLOps
* AIOps
* Data/Batch Processing
* CI/CD Pipelines

