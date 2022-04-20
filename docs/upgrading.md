
# Upgrading

Breaking changes  typically (sometimes we don't realise they are breaking) have "!" in the commit message, as per
the [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/#summary).

## Upgrading to v3.4

### feat!: Add entrypoint lookup. Fixes #8344

Affected if:

* Using the Emissary executor.
* Used the `args` field for any entry in `images`.

This PR automatically looks up the command and entrypoint. The implementation for config look-up was incorrect (it
allowed you to specify `args` but not `entrypoint`). `args` has been removed to correct the behaviour.

If you are incorrectly configured, the workflow controller will error on start-up. 

#### Actions

You don't need to configure images that use v2 manifests anymore. You can just remove them (e.g. argoproj/argosay:v2):

```bash
% docker manifest inspect argoproj/argosay:v2
...
"schemaVersion": 2,
...
```

For v1 manifests (e.g. docker/whalesay:latest):

```bash
% docker image inspect -f '{{.Config.Entrypoint}} {{.Config.Cmd}}' docker/whalesay:latest
[] [/bin/bash]
````

```yaml
images:
  docker/whalesay:latest:
    cmd: [/bin/bash]
```

## feat: Fail on invalid config. (#8295)

The workflow controller will error on start-up if incorrectly configured, rather than silently ignoring
mis-configuration.

```
Failed to register watch for controller config map: error unmarshaling JSON: while decoding JSON: json: unknown field \"args\"
```

## Upgrading to v3.3

### [662a7295b](https://github.com/argoproj/argo-workflows/commit/662a7295b) feat: Replace `patch pod` with `create workflowtaskresult`. Fixes #3961 (#8000)

The PR changes the permissions that can be used by a workflow to remove the `pod patch` permission. 

See [workflow RBAC](workflow-rbac.md) and [#8013](https://github.com/argoproj/argo-workflows/issues/3961).

### [06d4bf76f](https://github.com/argoproj/argo-workflows/commit/06d4bf76f) fix: Reduce agent permissions. Fixes #7986 (#7987)

The PR changes the permissions used by the agent to report back the outcome of HTTP template requests. The permission `patch workflowtasksets/status` replaces `patch workflowtasksets`, for example:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: agent
rules:
  - apiGroups:
      - argoproj.io
    resources:
      - workflowtasksets/status
    verbs:
      - patch
```

Workflows running during any upgrade should be give both permissions.

See [#8013](https://github.com/argoproj/argo-workflows/issues/8013).

### feat!: Remove deprecated config flags

This PR removes the following configmap items -

- executorImage (use executor.image in configmap instead)
  e.g.
  Workflow controller configmap similar to the following one given below won't be valid anymore:

  ```yaml
  apiVersion: v1
  kind: ConfigMap
  metadata:
    name: workflow-controller-configmap
  data:
    ...
    executorImage: argoproj/argocli:latest
    ...
  ```

  From now and onwards, only provide the executor image in workflow controller as a command argument as shown below:

  ```yaml
  apiVersion: v1
  kind: ConfigMap
  metadata:
    name: workflow-controller-configmap
  data:
    ...
    executor: |
      image: argoproj/argocli:latest
    ...
  ```

- executorImagePullPolicy (use executor.imagePullPolicy in configmap instead)
  e.g.
  Workflow controller configmap similar to the following one given below won't be valid anymore:

  ```yaml
  data:
    ...
    executorImagePullPolicy: IfNotPresent
    ...
  ```

  Change it as shown below:

  ```yaml
  data:
    ...
    executor: |
      imagePullPolicy: IfNotPresent
    ...
  ```

- executorResources (use executor.resources in configmap instead)
  e.g.
  Workflow controller configmap similar to the following one given below won't be valid anymore:

  ```yaml
  data:
    ...
    executorResources:
      requests:
        cpu: 0.1
        memory: 64Mi
      limits:
        cpu: 0.5
        memory: 512Mi
    ...
  ```

  Change it as shown below:

  ```yaml
  data:
    ...
    executor: |
      resources:
        requests:
          cpu: 0.1
          memory: 64Mi
        limits:
          cpu: 0.5
          memory: 512Mi
    ...
  ```

### [fce82d572](https://github.com/argoproj/argo-workflows/commit/fce82d5727b89cfe49e8e3568fff40725bd43734) feat: Remove pod workers (#7837)

This PR removes pod workers from the code, the pod informer directly writes into the workflow queue. As a result the `--pod-workers` flag has been removed. 

### [93c11a24ff](https://github.com/argoproj/argo-workflows/commit/93c11a24ff06049c2197149acd787f702e5c1f9b) feat: Add TLS to Metrics and Telemetry servers (#7041)

This PR adds the ability to send metrics over TLS with a self-signed certificate. In v3.5 this will be enabled by default, so it is recommended that users enable this functionality now.

### [0758eab11](https://github.com/argoproj/argo-workflows/commit/0758eab11decb8a1e741abef3e0ec08c48a69ab8) feat(server)!: Sync dispatch of webhook events by default

This is not expected to impact users.

Events dispatch in the Argo Server has been change from async to sync by default. This is so that errors are surfaced to
the client, rather than only appearing as logs or Kubernetes events. It is possible that response times under load are
too long for your client and you may prefer to revert this behaviour.

To revert this behaviour, restart Argo Server with `ARGO_EVENT_ASYNC_DISPATCH=true`. Make sure that `asyncDispatch=true`
is logged.

### [bd49c6303](https://github.com/argoproj/argo-workflows/commit/bd49c630328d30206a5c5b78cbc9a00700a28e7d) fix(artifact)!: default https to any URL missing a scheme. Fixes #6973

HTTPArtifact without a scheme will now defaults to https instead of http

user need to explicitly include a http prefix if they want to retrieve HTTPArtifact through http

### chore!: Remove the hidden flag `--verify` from `argo submit`.

The hidden flag `--verify` has been removed from `argo submit`. This is a internal testing flag we don't need anymore.

## Upgrading to v3.2

### [e5b131a33](https://github.com/argoproj/argo-workflows/commit/e5b131a33) feat: Add template node to pod name. Fixes #1319 (#6712)

This add the template name to the pod name, to make it easier to understand which pod ran which step. This behaviour can be reverted by setting `POD_NAMES=v1` on the workflow controller. 

### [be63efe89](https://github.com/argoproj/argo-workflows/commit/be63efe89) feat(executor)!: Change `argoexec` base image to alpine. Closes #5720 (#6006)

Changing from Debian to Alpine reduces the size of the `argoexec` image, resulting is faster starting workflow pods, and it also reduce the risk of security issues. There is not such thing as a free lunch. There maybe other behaviour changes we don't know of yet.

Some users found this change prevented workflow with very large parameters from running. See [#7586](https://github.com/argoproj/argo-workflows/issues/7586)

### [48d7ad3](https://github.com/argoproj/argo-workflows/commit/48d7ad36c14e4a50c50332d6decd543a1b732b69) chore: Remove onExit naming transition scaffolding code (#6297)

When upgrading from `<v2.12` to `>v3.2` workflows that are running at the time of the upgrade and have `onExit` steps _may_ experience the `onExit` step running twice. This is only applicable for workflows that began running before a `workflow-controller` upgrade and are still running after the upgrade is complete. This is only applicable for upgrading from `v2.12` or earlier directly to `v3.2` or later. Even under these conditions, duplicate work may not be experienced.

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

### [be63efe89](https://github.com/argoproj/argo-workflows/commit/e6fa41a) feat(controller): Expression template tags. Resolves #4548 & #1293 (#5115)

This PR introduced a new expression syntax know as "expression tag template". A user has reported that this does not
always play nicely with the `when` condition syntax (Goevaluate).

This can be resolved using a single quote in your when expression:

```
when: "'{{inputs.parameters.should-print}}' != '2021-01-01'"
```

[Learn more](https://github.com/argoproj/argo-workflows/issues/6314)

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
