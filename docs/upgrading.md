# Upgrading

Breaking changes  typically (sometimes we don't realise they are breaking) have "!" in the commit message, as per
the [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/#summary).

## Upgrading to v3.2

### [be63efe89](https://github.com/argoproj/argo-workflows/commit/be63efe89) feat(executor)!: Change `argoexec` base image to alpine. Closes #5720 (#6006)

Changing from Debian to Alpine reduces the size of the `argoexec` image, resulting is faster starting workflow pods, and it also reduce the risk of security issues. There is not such thing as a free lunch. There maybe other behaviour changes we don't know of yet. 

## Upgrading to v3.1

### [3fff791e4](https://github.com/argoproj/argo-workflows/commit/3fff791e4ef5b7e1de82ccb36cae327e8eb726f6) build!: Automatically add manifests to `v*` tags (#5880)

The manifests in the repository on the tag will no longer contain the image tag, instead they will contain `:latest`.

* You must not get your manifests from the Git repository, you must get them from the release notes.
* You must not use the `stable` tag. This is defunct, and will be removed in v3.1.

### [ab361667a](https://github.com/argoproj/argo-workflows/commit/ab361667a) feat(controller) Emissary executor.  (#4925)

The Emissary executor is not a breaking change per-se, but it is brand new so we would not recommend you use it by default yet. Instead, we recommend you test it out on some workflows using [config map configuration](https://github.com/argoproj/argo-workflows/blob/master/docs/workflow-controller-configmap.yaml#L125).

```yaml
# Specifies the executor to use.
#
# You can use this to:
# * Tailor your executor based on your preference for security or performance.
# * Test out an executor without committing yourself to use it for every workflow.
#
# To find out which executor was actually use, see the `wait` container logs.
#
# The list is in order of precedence; the first matching executor is used.
# This has precedence over `containerRuntimeExecutor`.
containerRuntimeExecutors: |
  - name: emissary
    selector:
      matchLabels:
        workflows.argoproj.io/container-runtime-executor: emissary
```


## Upgrading to v3.0

### [defbd600e](https://github.com/argoproj/argo-workflows/commit/defbd600e37258c8cdf30f64d4da9f4563eb7901) fix: Default ARGO_SECURE=true. Fixes #5607 (#5626)

The server now starts with TLS enabled by default if a key is available. The original behaviour can be configured with `--secure=false`.

If you have an ingress, you may need to add the appropriate annotations:(varies by ingress):

```yaml
alb.ingress.kubernetes.io/backend-protocol: HTTPS
nginx.ingress.kubernetes.io/backend-protocol: HTTPS
```

### [01d310235](https://github.com/argoproj/argo-workflows/commit/01d310235a9349e6d552c758964cc2250a9e9616) chore(server)!: Required authentication by default. Resolves #5206 (#5211)

To login to the user interface, you must provide a login token. The original behaviour can be configured with `--auth-mode=server`.

### [f31e0c6f9](https://github.com/argoproj/argo-workflows/commit/f31e0c6f92ec5e383d2f32f57a822a518cbbef86) chore!: Remove deprecated fields (#5035)

Some fields that were deprecated in early 2020 have been removed.  

| Field | Action |
|---|---|
| template.template and template.templateRef | The workflow spec must be changed to use steps or DAG, otherwise the workflow will error. |
| spec.ttlSecondsAfterFinished | change to `spec.ttlStrategy.secondsAfterCompletion`, otherwise the workflow will not be garbage collected as expected. |

To find impacted workflows:

```bash
kubectl get wf --all-namespaces -o yaml | grep templateRef
kubectl get wf --all-namespaces -o yaml | grep ttlSecondsAfterFinished
```

### [c8215f972](https://github.com/argoproj/argo-workflows/commit/c8215f972502435e6bc5b232823ecb6df919f952) feat(controller)!: Key-only artifacts. Fixes #3184 (#4618)

This change is not breaking per-se, but many users do not appear to aware of [artifact repository ref](artifact-repository-ref.md), so check your usage of that feature if you have problems.
