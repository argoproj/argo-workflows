# Upgrading Guide

Breaking changes  typically (sometimes we don't realise they are breaking) have "!" in the commit message, as per
the [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/#summary).

## Upgrading to v3.7

See also the list of [new features in 3.7](new-features.md).

### Deprecations

The following features are deprecated and will be removed in a future verison of Argo Workflows:

* The Python SDK is deprecated, we recommend migrating to [Hera](https://github.com/argoproj-labs/hera)
* `schedule` in CronWorkflows, `podPriority`, `mutex` and `semaphore` in Workflows and WorkflowTemplates.

For more information on how to migrate these see [deprecations](deprecations.md)

### Removed Docker Hub Image Publishing

Pull Request [#14457](https://github.com/argoproj/argo-workflows/pull/14457) removed pushing to docker hub.
Argo Workflows exclusively uses quay.io now.

### Made Parameter Value Overriding Consistent

Pull Request [#14462](https://github.com/argoproj/argo-workflows/pull/14462) made parameter value overriding consistent.
This fix changes the priority in which the values are processed, meaning that a Workflow argument will now take priority.
For more details see the example provided [here](https://github.com/argoproj/argo-workflows/issues/14426)
