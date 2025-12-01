# Upgrading Guide

Breaking changes  typically (sometimes we don't realise they are breaking) have "!" in the commit message, as per
the [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/#summary).

## Upgrading to v3.7

See also the list of [new features in 3.7](new-features.md).

For upgrades to older versions of Argo Workflows, please change to the documentation for the version of interest.

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

## Upgrading to 4.0

### Logging levels

The logging levels available have been reduced to `debug`, `info`, `warn` and `error`.
Other levels will be mapped to their equivalent if you use them, although they were previously undocumented.

### Full CRDs

The [official release manifests](installation.md#official-release-manifests) now default to using CRDs with full validation information.
This enables using [Validating Admission Policy](https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/) and `kubectl explain ...` on Argo CRDs.

Existing installations using the [minimal CRDs](https://github.com/argoproj/argo-workflows/tree/main/manifests/base/crds/minimal) will continue to work, but you'll be unable to use features that rely on CRD validation information.

Use the following command to selectively apply the full CRDs for an existing installation:

```bash
kubectl apply --server-side --kustomize https://github.com/argoproj/argo-workflows/manifests/base/crds/full?ref=v4.0.0
```
