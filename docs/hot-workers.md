# Hot Workers

> v4.2 and after

!!! warning "Experimental"
    Hot workers are an experimental feature.
    They are off by default and may change in incompatible ways in future releases.

Hot workers run container and script templates on a pool of long-lived worker pods, instead of creating a pod for each run.
A template opts in with `pool:`.
On this page, a task is what a pod would run for one attempt of a pooled node, whether the node is a step or a DAG task, and `{{node.taskName}}` is its name.
Each worker pod runs one task at a time, one after another.

Use hot workers for many short tasks, where creating, scheduling and starting a pod takes a large share of each task's time, for example a fan-out of thousands of tasks of a few seconds each, or a frequent `CronWorkflow` whose tasks are quick.
Keep long or resource-heavy tasks, and tasks that need the isolation of a pod of their own, on ordinary pods.
A pool's worker pods share files and credentials between the tasks they run, so read [isolation and security](#isolation-and-security) before you pool a template.

Hot workers run only on Kubernetes, within one cluster, only on Linux nodes, and only for container and script templates.

## How hot workers work

When a pooled task is ready to run, the controller creates a `WorkflowTaskSet` holding the task instead of a pod.
The first task for a pool creates the pool: a Deployment of worker pods, built from the template the same way a pod for that template would be.
A worker pod claims a task by creating the task's `WorkflowTaskResult`, runs the task, and reports its outputs through that result, as a pod does.
The controller deletes the task's `WorkflowTaskSet` once the node has finished.
A pool with no tasks is deleted after its retention period.

Every worker pod has two containers:

* `main` runs your template's image with `argoexec emissary --loop` as its entry point, which runs each task's command in turn.
* `worker` runs `argoexec worker`, which claims tasks, stages input artifacts, saves output artifacts and logs, and reports results.
  Artifact repository credentials are mounted only in `worker`.

With the default pod layout, the pool's pods also have an `init` init container that runs `argoexec install`, which only copies `argoexec` into the pod for `main`.
With the [init-less pod layout](initless-pod.md), `main` gets `argoexec` from an image volume instead.
Artifact plugins run as sidecars of the pool's pods and are shared by its tasks.

## Enabling hot workers

Enabling hot workers takes three steps:

1. Give the workflow controller access to Deployments, with the [hot workers RBAC component](#controller-rbac).
2. Set `hotWorkers.enabled` in the `workflow-controller-configmap`:

    ```yaml
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: workflow-controller-configmap
    data:
      hotWorkers: |
        enabled: true
    ```

3. Restart the workflow controller.

The controller reads `hotWorkers.enabled` only when it starts, and runs with that value until it restarts: both dispatching pooled tasks and the pool controller, which creates and deletes the pools' Deployments, follow it.
Until you restart the controller, pooled nodes fail as with hot workers disabled.
The controller logs a warning when `hotWorkers.enabled` changes without a restart.

If the controller can't manage Deployments when it starts with hot workers enabled, it doesn't run the pool controller, and every pooled node waits `Pending` with the message `pool "<name>" is degraded: the workflow controller isn't allowed to manage Deployments, which hot workers need: install the hot workers RBAC component and restart the controller`.

With hot workers disabled, a node whose template is pooled fails with `Error` and the message `template '<name>' has a pool, but hot workers are disabled: set hotWorkers.enabled in the workflow controller configmap and restart the controller`.
It never runs as an ordinary pod instead.
Other templates are unaffected.

The [workflow controller configmap reference](workflow-controller-configmap.md#hotworkersconfig) lists the rest of the `hotWorkers` settings, with their defaults and limits:

* `defaultRetentionSeconds`, a pool's [retention](#retention) when its template's `pool` doesn't set `retentionSeconds`;
* `terminationGracePeriodSeconds` and `releaseReserveSeconds`, which decide when a terminating worker [hands its task back](#failures-releases-and-retries);
* `leaseSlackSeconds`, how long the controller waits for a worker past its task's [deadline](#deadlines) before ending the task itself, and after ending a running task, as when you [stop the workflow](#stop-terminate-retry-and-delete), before treating the task as stopped.

A setting outside its limits makes the workflow controller exit with an error naming the setting, both when it starts and when it reads the changed configmap.

### Turning hot workers off

Before you turn hot workers off, let every workflow that uses pooled templates finish.
Turning the flag off while pooled tasks are queued or running isn't supported.
Once no pooled work remains, set `enabled: false`, restart the controller, which runs hot workers until it restarts, and delete any of its pools that are left.
Select its pools by its instance ID, so that other controllers' pools stay:

```bash
# A controller with an instance ID
kubectl delete deployments --all-namespaces -l 'workflows.argoproj.io/pool-hash,workflows.argoproj.io/controller-instanceid=<instance ID>'
# A controller without one
kubectl delete deployments --all-namespaces -l 'workflows.argoproj.io/pool-hash,!workflows.argoproj.io/controller-instanceid'
```

For a namespaced controller, use `-n <namespace>` instead of `--all-namespaces`.

## Pooling a template

Add `pool:` to a container or script template:

```yaml
- name: square
  pool:
    name: python-squares    # optional, see "How pools are formed"
    initialReplicas: 2      # optional, default 1
    retentionSeconds: 900   # optional, default hotWorkers.defaultRetentionSeconds
  inputs:
    parameters:
      - name: number
  script:
    image: python:alpine3.23
    command: [python]
    source: |
      print({{inputs.parameters.number}} ** 2)
```

| Field | Meaning |
| --- | --- |
| `name` | The pool's name. It must be a DNS label of at most 35 characters. It may hold a variable, such as `{{workflow.parameters.pool}}`, and is then checked once substituted. See [how pools are formed](#how-pools-are-formed). |
| `initialReplicas` | How many worker pods a new pool starts with, at least `1`. It applies only when the pool is created; to change the size of an existing pool, [scale it](#scaling). |
| `retentionSeconds` | How long the pool is kept once no task uses it. The value of the most recently dispatched task for the pool applies. |

See the [hot workers example](https://raw.githubusercontent.com/argoproj/argo-workflows/main/examples/hot-workers.yaml) for a complete workflow.

Everything else about the template works as it does for a pod: parameters, artifacts, `outputs.result`, exit codes, retries, memoization, synchronization, parallelism, metrics, `archiveLogs` and exit handlers.
The [differences from pods](#differences-from-pods) list every exception.

### `templateDefaults`

`pool` can also be set in `templateDefaults`, in the workflow or in the controller's [default workflow spec](default-workflow-specs.md).
An inherited `pool` pools only container and script templates; other template types ignore it, as they ignore an inherited `timeout`.
Setting `pool` directly on any other template type is rejected.

### What a pooled template can't use

These are rejected when you lint or submit the workflow, or when the task is dispatched if they only appear once the templates and defaults are merged:

| Rejected | Why |
| --- | --- |
| `pool` on a template other than container or script | Only container and script templates can be pooled. `containerSet`, resource, data, HTTP, plugin and suspend templates can't. |
| `daemon` | A worker runs one task at a time, and a daemon runs on after its node is ready. |
| `{{pod.name}}` | A pooled task has no pod of its own. Use `{{node.taskName}}`, the name its pod would have had. |
| `initContainers`, `sidecars` | They would run once per worker pod and be shared by every task of the pool, not run per task. |
| `podSpecPatch`, on the template or the workflow | It would patch the pool's pod template, which every task of the pool shares. |
| A mount of one of the workflow's `volumeClaimTemplates` | The claim is deleted with its workflow, while the pool serves other workflows. |
| `retryStrategy.affinity` | It changes the pod spec, and so the pool, on every attempt. |
| An input artifact `path` that is an ancestor of a volume mount | Staging the artifact would clear the mounted volume. |
| An input artifact `path` that is the container's `workingDir`, or holds it | The container runtime creates the `workingDir`, and a pooled task's input path must not exist (see [isolation and security](#isolation-and-security)), so every task would fail. Put the artifact beneath the `workingDir`. |
| `ARGO_DEBUG_PAUSE_BEFORE`, `ARGO_DEBUG_PAUSE_AFTER` in the container's `env` | A pool's worker pods run every task of the pool, so they can't pause for one. |
| `restartPolicy`, `restartPolicyRules` | The pool's `main` container restarts with its pod. |
| `livenessProbe`, `readinessProbe` | Probes run between tasks too, and a readiness probe would mark idle worker pods unready. |
| `startupProbe` | It runs once per worker pod, before any task. |
| `lifecycle` | Hooks run once per worker pod, not per task. |

A rejected field in the controller's `workflowDefaults`, such as `spec.podSpecPatch`, `spec.retryStrategy.affinity`, or a `spec.templateDefaults` that adds a rejected field to pooled templates, makes every workflow that uses a pooled template fail its checks.
The message names the field, but nothing warns you about it when you change the configmap.

## Pools

### How pools are formed

A pool is identified by its namespace and a hash of everything its worker pods are built from:

* the pool's name;
* the template's pod-level settings, such as its image, `imagePullPolicy`, `resources`, `workingDir`, `ports`, `securityContext`, volumes and volume mounts, `envFrom` and `valueFrom` environment variables, `metadata`, node selector, affinity, tolerations, scheduler, priority class, host aliases, pod resources and resource claims;
* the pod-level settings of the workflow spec, such as `podMetadata`, `imagePullSecrets`, `hostNetwork`, `dnsPolicy` and the workflow volumes the template mounts;
* the service account and `automountServiceAccountToken`, the `executor` settings, the artifact repository, and the secrets and plugins of the template's artifacts;
* the controller's version, instance ID and pod layout, the executor image, the `executor` and `mainContainer` settings of the `workflow-controller-configmap`, and the `hotWorkers` grace period and release reserve.

Templates whose pool names and all of these settings match share a pool, even across templates, workflows and WorkflowTemplates.
Command, args, plain `value:` environment variables, inputs, outputs, script source, timeouts and retry settings are per task, so tasks that differ only in these share a pool.
Each distinct value of a templated setting gives a distinct pool: `{{workflow.name}}` in a pooled template's `metadata.labels` gives each workflow a pool of its own, which suits a large fan-out, and a pool name of `{{workflow.parameters.pool}}` lets the submitter choose.

Without `pool.name`, the pool is named after the template: `<workflowtemplate>-<template>` for a template from a WorkflowTemplate or ClusterWorkflowTemplate, through `templateRef` or `workflowTemplateRef`, and `<template>` for a template of the workflow's own, lowercased, with `.` replaced by `-` and cut to 35 characters.
Unrelated templates with the same name and settings share a default-named pool, so name your pools.

Pools belong to a namespace and never serve another namespace's tasks.
A template used from several namespaces, for example from a ClusterWorkflowTemplate, gets a pool in each, each with its own cold start and retention.

A pool's Deployment is named `<pool name>-<first 10 characters of the hash>`.
The Deployment, its pods and the pool's objects carry these labels:

| Label | On |
| --- | --- |
| `workflows.argoproj.io/pool-name` | The pool's Deployment and pods, and its tasks' `WorkflowTaskSets` and `WorkflowTaskResults` |
| `workflows.argoproj.io/pool-hash` | The same, with the pool's full hash |
| `workflows.argoproj.io/controller-instanceid` | The same, when the controller has an instance ID |
| `workflows.argoproj.io/worker-pod`, `workflows.argoproj.io/worker-pod-uid` | A claimed task's `WorkflowTaskResult`, naming the worker pod that claimed it |
| `workflows.argoproj.io/task-set-uid` | A claimed task's `WorkflowTaskResult`, holding the UID of the `WorkflowTaskSet` it claims |
| `workflows.argoproj.io/interrupt` | A claimed task's `WorkflowTaskResult`, once the controller has told its worker to stop the task |

`metadata` and `podMetadata` can't override the pool name, pool hash and instance ID labels on worker pods; a value you set for one of them is replaced without a warning.
Worker pods never carry the `workflows.argoproj.io/workflow` label, because they serve many workflows.

### Retention

The controller deletes a pool's Deployment once the pool has had no tasks for its retention period.
A task counts from when it is queued until its node has finished.
The retention in use is on the Deployment's `workflows.argoproj.io/pool-retention-seconds` annotation.
The controller tracks idle time in memory, so a controller restart starts the retention period again; it never deletes a pool early.

The first task after a pool is deleted waits for a new pool to start.
For a `CronWorkflow`, set `retentionSeconds` longer than its schedule's interval to keep the pool between runs.
With `retentionSeconds: 0`, the pool is deleted as soon as it has no tasks, which can happen between two steps of one workflow.

Upgrading the controller, or changing its `executor` or `mainContainer` settings, its pod layout or the `hotWorkers` grace period or release reserve, changes every pool's hash.
New tasks then go to new pools, each with a cold start, and the old pools are deleted once their retention passes.

### Scaling

The pool controller sets a pool's replicas only when it creates the Deployment, from `initialReplicas`, and never changes them afterwards.
Scale a pool by scaling its Deployment, with `kubectl scale` or with an autoscaler you configure for it, such as a `HorizontalPodAutoscaler`:

```bash
kubectl scale deployment python-squares-<hash> --replicas=5
```

The Deployment's name changes with the pool's hash, so an autoscaler you create covers one pool, and a pool created afresh starts again from `initialReplicas`.
Scaling down releases the tasks the removed pods can't finish within their grace period, and they run again elsewhere without using a retry (see [failures, releases and retries](#failures-releases-and-retries)).
A pool scaled to zero keeps its tasks waiting until you scale it up or a deadline passes.

The controller exports per-pool gauges, each with `namespace`, `pool_name` and `pool_hash` attributes:

* [`pool_backlog`](metrics.md#pool_backlog): tasks waiting for a worker.
* [`pool_claimed_tasks`](metrics.md#pool_claimed_tasks): tasks a worker is running.
* [`pool_ready_pods`](metrics.md#pool_ready_pods): ready worker pods.

A pool's saturation is `pool_claimed_tasks / pool_ready_pods`, since each worker pod runs one task at a time.
It can briefly exceed 1 while worker pods are being replaced.
A pool is reported while it has a task or a Deployment, or, when the pool controller isn't running, a worker pod that isn't being deleted.

### Pool health

A pooled node waits `Pending` until a worker claims it.
While its pool can't run tasks, the node's message says why:

* `pool "<name>" is degraded: <reason>` while the pool's Deployment can't be created, for example because admission or quota refuses it, or while none of the pool's pods is ready and they can't be created or can't start (an image that can't be pulled, a missing Secret or ConfigMap, a crash loop).
  The node waits until the pool recovers or a deadline passes.
  A pod whose creation is forbidden fails its node at once instead.
* `pool "<name>" can't be built: <reason>` when the pool can never be built, for example because a Deployment that isn't the pool's has its name, or the controller can't build the pool spec the task was dispatched with.
  The node fails with `Error`.

The pool controller also emits a `PoolDegraded` or `PoolUnbuildable` Warning event on the pool's Deployment, or on the `WorkflowTaskSet` if there is no Deployment, when the pool's health changes.

## RBAC

### Controller RBAC

The default installs give the workflow controller `get`, `list`, `watch`, `patch`, `delete` and `deletecollection` on `workflowtaskresults`, which pooled tasks need.
If you maintain your own controller roles, add these verbs.

Hot workers also need the controller to manage Deployments, which default installs don't grant.
Add the `manifests/components/hot-workers` Kustomize component to your install, from the same version, to add the rule to the controller's `argo-cluster-role` ClusterRole or its namespace install's `argo-role` Role:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - https://github.com/argoproj/argo-workflows/manifests/cluster-install?ref=v4.2.0
components:
  - https://github.com/argoproj/argo-workflows/manifests/components/hot-workers?ref=v4.2.0
```

If you don't use Kustomize, add this rule to the controller's role:

```yaml
- apiGroups:
    - apps
  resources:
    - deployments
  verbs:
    - get
    - list
    - watch
    - create
    - patch
    - delete
```

### The worker's service account

A pool runs as the template's or workflow's service account, as its pods would.
The `worker` container uses that account, unless you set `executor.serviceAccountName`, in which case it uses the executor's account and `main` keeps the pool's.

The worker's account needs more than the [executor's role](workflow-rbac.md), which has only `create` and `patch` on `workflowtaskresults`: it watches the namespace's `WorkflowTaskSets` for tasks and claims, releases and completes `WorkflowTaskResults`.
The quick-start manifests include this as the `hot-worker` Role, and bind it to the `default` service account:

```yaml title="manifests/quick-start/base/hot-worker-role.yaml"
--8<-- "manifests/quick-start/base/hot-worker-role.yaml:2"
```

The cluster and namespace installs don't include this Role.
If you use one of them, create the Role in each namespace whose workflows use pooled templates, for example in the `my-namespace` namespace:

```bash
kubectl apply -n my-namespace -f https://raw.githubusercontent.com/argoproj/argo-workflows/v4.2.0/manifests/quick-start/base/hot-worker-role.yaml
```

Then bind the role, or grant its verbs, to the pool's service account, or to the executor's account if you set `executor.serviceAccountName`, in the same namespace.
For example, for the `my-service-account` service account:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: hot-worker-my-service-account
  namespace: my-namespace
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: hot-worker
subjects:
  - kind: ServiceAccount
    name: my-service-account
```

The `worker` container checks these verbs at start, with `SelfSubjectAccessReviews`.
Without one of them, it exits with an error naming every verb it isn't allowed on each resource, and the account it runs as when the API server allows a `SelfSubjectReview` (Kubernetes 1.28 or later), so the pool's pods crash loop and the pool's waiting nodes show `pool "<name>" is degraded: CrashLoopBackOff: container "worker" keeps exiting`.

### Retrying pooled workflows

`argo retry` of a pooled workflow needs no permissions on `workflowtasksets` or `workflowtaskresults`.
A workflow completes only once its pooled `WorkflowTaskSets` are deleted, so the controller enqueues each pooled node the retry resets afresh, and deletes any `WorkflowTaskResult` its earlier attempt left.

## Differences from pods

A pooled task behaves as a pod would, except as this section describes.

### Isolation and security

The pool, not the task, is the isolation boundary:

* Files a task writes outside volumes, other than its declared outputs, stay in the worker pod and are visible to later tasks on it.
  Use an `emptyDir` volume for scratch space: the worker empties the template's and workflow's `emptyDir` volumes before and after each task.
  PVC and `hostPath` volumes keep their contents, as for pods, and are mounted by every worker pod of the pool.
* Every process a task starts is killed when the task ends, however it detached, so no process outlives its task.
  `main` gives up 30 seconds after the task's command exits on a process even `SIGKILL` can't end, such as one stuck on a hung volume: it saves none of the task's outputs, and its container restarts before the pod runs another task.
* Without `executor.serviceAccountName`, the task's code can use the worker's account, and so has the worker's rights over every pooled task in the namespace: it can read other tasks' templates, delete their claims, which kills those tasks and makes them run again, and claim other workflows' tasks.
  Set `executor.serviceAccountName` to keep these rights in `worker` alone, and give a pool's service account only what its tasks need.
* Tasks of different workflows, and of different users, share a pool when their templates match.
  Give templates that need separate pools different pool names or service accounts.

A task fails if one of its input artifact paths, or a declared output path, already exists outside a volume when it starts, for example a file in the image: running the task would otherwise delete the image's file, or report a file an earlier task left as this task's output.
Put inputs and outputs at paths that don't exist in the image, or in a volume.

In both pod layouts, `main` links each input artifact into place in its own filesystem, as it does for an ordinary pod in the [init-less pod layout](initless-pod.md#readonlyrootfilesystem-is-incompatible-with-init-less-input-artifacts).
So with `readOnlyRootFilesystem: true` in the container's `securityContext`, a task whose input artifact path is on the read-only root filesystem fails before its command runs.
Mount a writable volume, such as an `emptyDir`, at the directory that holds each such artifact's path; the worker then stages the artifact on the volume.

The worker empties `emptyDir` volumes, and removes the directories it staged a task's input artifacts in, as the `worker` container's user, which `executor.securityContext`, the pod's `securityContext` or the executor image sets.
If a task's command runs as another user, such as root, and creates directories there that this user can't write into, as `tar -x` or `cp -a` do when they keep a `0755` mode, the worker can't clean up after the task.
It then stops before it claims another task, and every time it restarts it fails to clean up again and stops, handing back any task it claimed, so the worker pod is stuck until you delete it.
To avoid this, run the task as the `worker` container's user, make the directories the task creates there writable by that user, for example group- or world-writable, or keep such files out of `emptyDir` volumes and input artifact directories.

### Environment variables

Plain `value:` environment variables are set per task.
A plain `PATH` is the task's: `main` finds the task's command on it, as the container runtime finds a pod's.
`valueFrom` and `envFrom` variables are part of the pool: they are read when the worker pod starts, so a rotated Secret or ConfigMap behind them isn't seen until the pool's pods are replaced, where a pod reads them every time it starts.
`fieldRef` variables, such as `metadata.name`, describe the worker pod.

`$(VAR)` references in a task's command, args and environment variables are expanded by `main`, not the kubelet, against `main`'s whole environment.
That includes the image's own `ENV` (so `$(PATH)` expands) and `valueFrom` variables listed later in the template.
As with the kubelet, unknown references are kept as they are and `$$` becomes `$`.

### Failures, releases and retries

A worker pod that crashes or is lost mid-task fails the node with `Error`, as a deleted pod does:

* the worker pod is deleted before its worker can hand the task back, as with a forced deletion, or fails, for example when the kubelet evicts it for node pressure: `worker pod <name> deleted` or `worker pod <name> failed: <reason>`;
* the worker pod finishes without failing before its worker hands the task back: `worker pod <name> exited`;
* its `worker` container restarts: `worker container of pod <name> restarted`.

A queued task whose `WorkflowTaskSet` is deleted, for example by hand, fails its node with `Error` and the message `WorkflowTaskSet <node ID>-task deleted`, since no worker can claim it.

The default `retryPolicy`, `OnFailure`, doesn't retry these; use `OnError` or `Always` to retry them.
A task that fails to stage its inputs or save its outputs is also `Error`, as a pod's init, wait or supervisor container failing is.

When `main` dies mid-task, for example because it was OOM-killed, the node fails with `Failed` and the message `main: the container exited before the task ended`, rather than the container's reason, such as `OOMKilled (exit code 137)`: the worker can't read the pod's status.

A worker that is terminated gracefully, as when its pool is scaled down, its pod is deleted or its node is drained, hands its task back:

* a task whose inputs the worker is still staging, or that `main` hasn't yet started, is released at once;
* a task `main` is running that finishes before the grace period less `hotWorkers.releaseReserveSeconds` completes as usual, and so does a task whose outputs the worker is saving;
* otherwise the worker stops the task and releases it.

Another worker runs a released task again from the start.
When the worker stops a task, for a release or otherwise, `main` sends the task's processes `SIGTERM`, then `SIGKILL` to any left after half of `hotWorkers.releaseReserveSeconds`.

A released task doesn't use a retry, even without a `retryStrategy`.
Its node stays `Running` while it waits for another worker.
A task whose `WorkflowTaskResult` is deleted while its `WorkflowTaskSet` remains, for example by hand, is also run again rather than failed.

A workflow's `podDisruptionBudget` doesn't cover its pooled tasks, because worker pods don't carry its labels: a drain releases their tasks, and they run again.

### Deadlines

Pooled tasks are bounded by the same deadlines as pods, from the same start points:

| Deadline | Pooled task |
| --- | --- |
| Template `activeDeadlineSeconds` | Counted from the claim, as a pod's is from its start, including the time to stage inputs. The task fails with `Pod was active on the node longer than the specified deadline`. |
| `timeout` | Counted from the node's start, as for a pod. A task still queued fails with `timeout`; a task running, or released and waiting, fails with `Step exceeded its deadline`, where a pod says `Pod was active on the node longer than the specified deadline`. |
| `pendingTimeout` | Covers only the wait before the first claim. It doesn't apply again after a release. |
| Workflow `activeDeadlineSeconds`, `retryStrategy.backoff.maxDuration` | As for a pod. |

The worker enforces a running task's deadlines itself, as the kubelet does for a pod, and kills the task when one passes.
`activeDeadlineSeconds` runs from the claim's creation timestamp on the API server, which has one-second precision, so it can end the task up to about a second early, plus any clock skew between the worker's node and the API server.
A pod merges its deadlines into one `activeDeadlineSeconds`, so a pod past its workflow deadline can show either message, but a pooled task always shows the message of the deadline that passed.

A queued task's `timeout` and `pendingTimeout` are checked only while its workflow is reconciled, and not while the workflow is suspended, as for pending pods.
After a release, the task is bounded by its `timeout`, its `retryStrategy.backoff.maxDuration` and the workflow's deadline, and its `activeDeadlineSeconds` counts again from the next claim.

If a running task's worker hasn't finished `hotWorkers.leaseSlackSeconds` after the task's deadline, the controller ends the task itself.
A wedged worker running a task with no deadline keeps it, as a hung pod with no deadline does.

### Logs, events and links

Pooled tasks have no live logs, events or pod links.
Their archived logs work: enable [`archiveLogs`](configure-archive-logs.md) to keep them.

* The UI shows a pooled node's task name and worker pod as separate rows, and links to its archived logs.
* `argo logs <workflow>`, also with `--selector`, prints a note for each pooled task saying where its archived logs are, with the `argo cp` command that fetches them once they exist.
  `argo cp` downloads through the Argo Server, so it fails when the CLI talks to Kubernetes directly, without one.
  `argo logs <workflow> <name>`, where `<name>` names only pooled tasks by task name or worker pod, prints the notes and exits non-zero.
* The Argo Server's log API, and the UI's view of all of a workflow's logs, show nothing for pooled tasks.

`argo get` shows a pooled node's worker pod once a worker has claimed it, and otherwise its task name, marked `(queued)` while it waits, or `(unclaimed)` or `(released)` if it ended before a worker held it.

Artifacts and logs are stored under the key they would have as a pod: the default key format, `{{workflow.name}}/{{node.taskName}}`, gives each attempt its own key, and `{{pod.name}}` in a repository's key format resolves to the task name.

### Other differences

* `{{steps.<name>.ip}}`, `{{tasks.<name>.ip}}`, the node's host node name and `fieldRef` environment variables refer to the worker pod.
* Each `{{workflow.failures}}` entry has a `taskName`, the node's `{{node.taskName}}`, and its `podName` is the worker pod, or empty for a pooled node no worker holds: one never claimed, or one a worker released.
* A pooled node's `resourcesDuration` stays zero.
* A pooled task's `WorkflowTaskSet` is named `<node ID>-task`.
  If another workflow in the namespace is named exactly that and runs HTTP or plugin templates, whichever `WorkflowTaskSet` takes the name first keeps it: if the other workflow's holds it, the pooled node fails rather than use it, and if the pooled task's holds it, the other workflow's HTTP and plugin tasks fail, saying so.

## Stop, terminate, retry and delete

`argo stop`, `argo terminate` and the workflow deadline end pooled nodes as they end pods.
Queued tasks are dropped, and their nodes fail at once.
Running tasks are killed, and their nodes fail with pods' message at once, but each stays unfinished, holding its mutex or semaphore and keeping the workflow from completing, until its worker confirms the task stopped, usually within seconds, or `hotWorkers.leaseSlackSeconds` passes.
As for pods, the enclosing template fails at once, so the workflow's exit handler may start before then.
`argo stop` lets exit handlers run; `argo terminate` kills them too.

After `argo retry` of a terminated node, the old task can briefly overlap the new attempt until its worker sees that its claim is gone.
The old task can never write the new attempt's result.

A workflow completes only once all its pooled `WorkflowTaskSets` are deleted.
If the controller can't delete them, the workflow stays running with the message `waiting to delete pooled WorkflowTaskSets: <error>` and a `WorkflowPooledTaskSetsRemain` Warning event, until the deletes succeed or you delete the `WorkflowTaskSets` yourself.

Deleting a workflow kills its running pooled tasks: its `WorkflowTaskSets` and `WorkflowTaskResults` are garbage collected with it, and a worker whose claim is deleted stops its task and reports nothing.

## Upgrading and rolling back

Install the v4.2 or later CRDs before you use `pool:`.
With the full CRDs of an earlier version, the API server drops `pool` from your templates, and they run as ordinary pods.
With an earlier `WorkflowTaskSet` CRD, the API server drops the task's pool spec, and the node fails with `the WorkflowTaskSet was stored without spec.pool or spec.dispatch; install the current CRDs`.

A controller upgrade leaves running pooled tasks on their pools.
Tasks dispatched by the new controller go to new pools, as [retention](#retention) describes.
A controller never creates or changes the pool of a task that another controller version dispatched.
The pool's Deployment, if it exists, keeps running until its retention passes, and isn't created again once deleted.
The new controller replaces each queued task of an earlier version, which no worker has claimed, with one for a pool of its own version, so the task waits for that pool's cold start even when the earlier version's pool has idle worker pods.
A task that a worker has already claimed finishes on that worker's pod.
If a worker claims a task just as the new controller replaces it, the claim is interrupted, and the node fails with `Error` and the message `a worker claimed the task as it was moved to this controller version's pool` once the task stops, for a retry to run it afresh; a worker that hadn't yet started the task gives it up, and it is replaced as if unclaimed.
A task a worker released stays queued for its earlier pool, and if that pool's `Deployment` is gone, its node stays `Pending` with `pool "<name>" is degraded: the pool spec is for controller version "<version>", and this controller, "<version>", builds pools only for its own version`.

Before you roll the controller back to a version without hot workers, let every workflow that uses pooled templates finish, then delete the pools as in [turning hot workers off](#turning-hot-workers-off).

## Troubleshooting

* A pooled node fails with `template '<name>' has a pool, but hot workers are disabled`: enable hot workers and restart the controller.
* Pooled nodes stay `Pending` with `pool "<name>" is degraded: ...`: fix the reason, such as the image, quota or controller RBAC.
* Pooled nodes stay `Pending` with `pool "<name>" is degraded: CrashLoopBackOff: container "worker" keeps exiting`: check the `worker` container's logs or termination message.
  A message such as `the worker isn't allowed to watch workflowtasksets; create, patch workflowtaskresults in namespace argo` means the account it runs as lacks those verbs of the [`hot-worker` Role](#the-workers-service-account); the message names the account when the API server allows a `SelfSubjectReview`.
* Pooled nodes stay `Pending` with `pool "<name>" is degraded: CrashLoopBackOff: container "main" keeps exiting`, and `main` logs `emissary --loop runs only on Linux`: the pool's pods run on Windows nodes.
  Schedule the template on Linux nodes, for example with a `nodeSelector` of `kubernetes.io/os: linux`.
* A worker pod's `worker` container keeps restarting, logging `permission denied` as it fails to empty a volume or remove a directory: a task left directories the worker can't remove.
  Delete the pod, and see [isolation and security](#isolation-and-security) for how to avoid it.
* A task fails because a path already exists: an input artifact or declared output path is in the image, or outside a volume; see [isolation and security](#isolation-and-security).
* A workflow doesn't complete, with the message `waiting to delete pooled WorkflowTaskSets`: fix what stops the controller deleting them, such as its RBAC or an admission webhook, or delete them yourself.

## Limitations

Hot workers don't support:

* Windows nodes: `main` runs only on Linux;
* `containerSet`, daemon, resource, data, HTTP and plugin templates;
* `initContainers`, `sidecars` and `podSpecPatch` on pooled templates;
* live logs, and logs of pooled tasks through the Argo Server's log API;
* a different `workingDir` per task of one pool: templates that differ only in `workingDir` get separate pools;
* creating a pool before its first task, or scaling an idle pool to zero: an idle pool keeps its pods until its retention passes, and is then deleted;
* autoscaling: the pool controller doesn't create autoscalers for its pools;
* disruption budgets for pooled tasks;
* a task whose command, running as a user other than the `worker` container's, creates directories the worker can't write into in an `emptyDir` volume or an input artifact directory: the worker pod is stuck until you delete it, as [isolation and security](#isolation-and-security) describes;
* a time limit on a workflow's wait to delete its pooled `WorkflowTaskSets`;
* a WorkflowTemplate or ClusterWorkflowTemplate that holds pooled templates for other workflows and sets a workflow-level field pooled templates reject: one with its own `spec.podSpecPatch` or `spec.retryStrategy.affinity` fails its lint if any of its templates is pooled, even when the pooled template is used only from other workflows through `templateRef`.
