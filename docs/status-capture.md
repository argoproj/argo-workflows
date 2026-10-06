# Pod Status Capture and Retained Pods

> v4.2 and after

## The problem this solves

`ARGO_POD_STATUS_CAPTURE_FINALIZER=true` makes the controller add the `workflows.argoproj.io/status` finalizer to each Workflow Pod, so that the Pod stays around until the controller has read its result.
Before v4.2 the finalizer came off on a timer: once a Pod was being deleted, the cleanup queue removed the finalizer two minutes after the Pod's last status transition, without checking whether the result had been saved.
When Workflow reconciliation fell behind, a Pod that had succeeded could be deleted before its result reached the Workflow status, and the Workflow then failed with `Error: pod deleted` ([#17024](https://github.com/argoproj/argo-workflows/issues/17024)).

From v4.2, a finished Pod keeps the finalizer until its result has been saved in the Workflow status under that Pod's own UID.
Turn the flag on if you have seen `pod deleted` errors on Pods that succeeded, or if losing a Pod's result during controller delays or restarts is not acceptable to you.
With the flag off, Pods carry no finalizer and nothing waits for them, but the controller still records Pod UIDs and performs some of the reads described in [Behavior with the flag off](#behavior-with-the-flag-off).

## What the controller records

Each node that ran as a Pod gets an optional status field, `capturedPodUID`, holding the UID of the Pod whose state produced the saved result.
The UID matters because a Pod can be replaced by a new Pod with the same name, for example after an automatic restart, and the saved result belongs to exactly one of them.
Cleanup removes the finalizer only from the Pod whose UID is recorded.
A node that is waiting for an automatic restart records the Pod it is retiring in `restartingPodUID` instead, and the controller deletes that Pod once the restart has been saved.
Daemon termination and the deletion of a completed Workflow's agent Pod are likewise carried out only after the corresponding decision has been saved in the Workflow status.
For nodes offloaded to a database, the result counts as saved only when the Workflow object references the saved node version; a database row alone is not enough.
A Pod whose Workflow is confirmed deleted, replaced by a newer Workflow of the same name, or being deleted is cleaned up by the existing orphan rule, which does not look for a saved result.

## Behavior with the flag off

Part of this mechanism runs whether or not the flag is set.

The controller writes `capturedPodUID` in both modes.
Installations that use the full CRDs need the v4.2 CRDs before the v4.2 controller starts; otherwise the API server drops the field from uncompressed node status.

Each Pod Add event costs one Pod GET and one Workflow GET against the API, plus a decode of the whole node map when the Workflow's nodes are compressed or offloaded.
The controller receives an Add for every existing Workflow Pod when it starts, so each restart produces a burst of reads proportional to the number of live Pods.
Later updates of a running Pod add no reads until the Pod finishes, starts deleting, or changes its owner, labels, annotations or finalizers.

A Workflow that finishes while its status cannot be saved, for example because the offload database is unreachable, stays `Running` and is retried every 30 seconds until the write succeeds, keeping its parallelism slot and synchronization locks meanwhile.
Before v4.2 it was marked `Error`.
See [When the result cannot be saved](#when-the-result-cannot-be-saved) for the one case that still ends in `Error`.

The finalizer itself, and the check that a result is saved before the finalizer is removed, apply only with the flag on.

## When the result cannot be saved

If a Workflow finishes and its status is still above the size limit after compression, and node offloading is not configured, the controller marks the Workflow `Error` using the node data it last saved.
The results of the final reconciliation, including their `capturedPodUID` values, are not written.
Synchronization locks are released once that error has been saved; if that write fails as well, the locks stay held and the write is retried.
Parallelism slots follow the usual completion rules, including waiting for an outstanding artifact garbage collection finalizer.
Pods of that Workflow whose results were never saved keep their finalizer.
To release them, run the [inventory](#find-and-classify-retained-pods) and then [remove the finalizer yourself](#remove-a-finalizer-yourself) with the extra acknowledgement for an incomplete result.
Enabling offloading afterwards does not turn that `Error` into the `Succeeded` or `Failed` outcome the controller had computed in memory.

Any other storage failure, such as a SQL error, a network error or missing permissions, is retried with the original result; no new Pod event is needed.
The controller does not try to decide whether such an error is permanent, because an error that is not classified as transient can still be fixable.
Until storage is reachable again, the Workflow stays `Running` and keeps its parallelism slot and synchronization locks; restoring storage access is what lets it complete.
An outcome that was already saved is never replaced by this retry.

## Upgrade and rollback

Install the v4.2 full Workflow CRD before starting the v4.2 controller.
Keep the controller, Argo Server and CLI within the [supported version skew](releases.md#supported-version-skew).
Keep the database configuration, controller instance ID and namespace scope the same across the transition.
A controller rollout that reports success does not show that the API server kept the new field.
To check, read a Workflow that completed after the upgrade, decode its node status if it is compressed or offloaded, and confirm that the node carries the UID of the Pod that ran it.

The protection holds only while a v4.2 controller is the active leader.
An older controller does not pick up the new cleanup check from a CRD update, and as soon as an old replica becomes leader it goes back to removing finalizers on the timer.
An old replica that stays standby does nothing.
A normal rolling upgrade therefore has a window in which an old binary can become active again.
If you have Pods that are still waiting for their results to be saved and want them covered through the upgrade, use the order below.
Installations that run with the flag off do not need it, and it does not replace the general [rollout guidance](high-availability.md#deployment-rollout-strategy).
Stopping the controller does not stop Pods that are already running.

1. Pause new submissions and the relevant CronWorkflows with your usual controls.
   Pause operations that rewrite Workflow status, such as resume, suspend, retry and selected-node updates, until the Server and CLI are on matching versions.
2. Record the old controller image, the installed full CRD, the instance scope and the persistence configuration, without exporting credentials.
   Run the [inventory](#find-and-classify-retained-pods) of Pods that still carry the finalizer and keep its output.
3. Stop all old controller replicas for that scope and confirm that their processes have exited.
4. Apply the full CRDs from the same release as the new controller, then start only new replicas with status capture enabled.
5. Confirm the elected leader's image, inspect a few newly completed Workflows and their `capturedPodUID` values, and confirm that cleanup removed the finalizer from exactly those Pod UIDs.
   Classify any Pods left over from the old controller with the inventory, then resume submissions.

To roll back, pause submissions and let the v4.2 controller finish its active Workflows and the cleanup it still owes.
Then inspect every Pod that still carries the finalizer: fix temporary failures, keep the evidence for Pods the controller cannot verify, or make the explicit decision described at the end of this page.
If some of those Pods cannot be resolved, keep the v4.2 controller running or leave that scope stopped with its Pods and data in place, because an old controller removes their finalizers without checking.
Keep the v4.2 CRD after rolling back the controller; an older CRD drops the new status fields.
Older controllers, Argo Server and direct Kubernetes CLI versions can also drop `capturedPodUID` when they read and rewrite node status, because they do not know the field.
Resume, retry and selected-node updates rewrite node status; whole-Workflow stop and terminate use a narrow patch and do not.
When a command goes through Argo Server, the Server performs the write, so its version is the one that matters.
Once an old controller takes over a scope, the protection is gone for that scope.
To get it back, stop the old replicas, confirm the installed CRD, restart the new controller and repeat the inventory and `capturedPodUID` checks.
None of this brings back Pods that were already deleted, or UIDs that another writer already dropped.

## Completed Workflows from older controllers

Pods of Workflows that completed under an older controller have no `capturedPodUID`, so the v4.2 controller cannot tell whether the saved result came from the Pod it is looking at.
Only Pods that still carry the finalizer are affected, which means installations that already ran with the flag on.

The controller fills the field in by itself only for one narrow case: a completed Workflow with a single successful node, running an ordinary `container` template, whose Pod is still present and succeeded, and whose `ARGO_TEMPLATE` matches the stored template.
For that case the controller reads the Pod's container statuses, rebuilds the result, and accepts it only if exit code, message, finish time, progress, host and resource duration match the saved node exactly and the task result is explicitly recorded as synchronized.
Only `capturedPodUID` is added; the outcome and timestamps stay as they were.
The controller then reads the Workflow back and proceeds to its normal guarded cleanup only if the field was kept.

Every other completed Workflow from an older controller keeps its Pods held until you act.
That includes Workflows with more than one node, such as an ordinary successful two-step Workflow, as well as nodes with inputs or with outputs other than the exit code, memoized nodes, daemon nodes, stopped or terminated Workflows, deadlines, retries, automatic restarts, user sidecars or init containers, Pod spec patches, hooks, external template references, containers that restarted, and templates stored by reference in an environment source.
These Pods are not necessarily unrecoverable; the controller has no way to verify them, so it leaves the decision to you.

Two cases look wrong but are correct, and the controller leaves them alone.
A successful Pod can have a saved node `Error` because a later memoization write failed; the controller keeps that `Error` and does not retry the write or change it to `Succeeded`, even if the cache is healthy now.
A `WorkflowTaskResult` names a node and its Workflow but not the Pod UID that produced it, so it does not prove which Pod the saved data came from.
Workflows from older controllers that are still running are reconciled as normal, and nodes that already carry a `capturedPodUID` keep their meaning, including nodes whose task completion state was never set.

## Find and classify retained Pods

The helper `hack/status-capture.py` ships in the Argo Workflows source tree from v4.2.
Check out the tag that matches your controller version and run it from the repository root; it needs only the Python 3 standard library.
It reads saved JSON files and never contacts the API or the database; its tests run with `make test-status-capture`.
Work in a private directory with the credentials you normally use for the namespace.
The snapshots can contain parameter values, environment variables, artifact locations and output data, so keep them private and leave kubeconfig files and credentials out of anything you share.

Set `CAPTURE_NAMESPACE` and run:

```sh
umask 077
mkdir capture-evidence
kubectl -n "$CAPTURE_NAMESPACE" get pods -l workflows.argoproj.io/workflow -o json > capture-evidence/pods.json
kubectl -n "$CAPTURE_NAMESPACE" get workflows -o json > capture-evidence/workflows.json
python3 hack/status-capture.py inventory \
  --pods capture-evidence/pods.json \
  --workflows capture-evidence/workflows.json
```

The output lists every Pod that still carries the finalizer, with a `reason` for each.
It describes the snapshot files and nothing else: a Workflow missing from `workflows.json` only means the list did not contain it, not that the API returned NotFound, and no `reason` is permission to remove a finalizer.

| `reason` | Meaning | What to do |
|---|---|---|
| `matching-receipt-in-snapshot` | The node's `capturedPodUID` is this Pod. | Nothing; once the controller has re-read the live objects, normal cleanup, any PodGC delay and retries apply. |
| `result-or-task-pending` | The node is not in a terminal phase, or its task result is not yet marked as synchronized. | Let reconciliation finish and restore whatever it is waiting for, such as the database or the executor; do not force completion. |
| `active-workflow-awaiting-capture` | The node is finished but its Workflow is still running and the node has no `capturedPodUID`. | Let the Workflow finish and run the inventory again. |
| `pod-not-terminal` | The Pod is not `Succeeded` or `Failed`. | Wait for it to finish, or stop the Workflow. |
| `persisted-restart-disposition` | The node is pending and names this Pod in `restartingPodUID`. | Let the controller delete the retired Pod. |
| `completed-storage-error-without-capture` | The Workflow ended with the oversize storage `Error` described above and this node has no `capturedPodUID`. | Save the Pod, Workflow and task results first; the saved node data is incomplete, and releasing the Pod needs the extra acknowledgement described below. |
| `legacy-no-receipt` | The Workflow completed and the node is terminal but has no `capturedPodUID`. | Check the `legacyClass` hint; if the controller cannot capture it by itself, keep the evidence or make the explicit decision below. |
| `capture-identity-conflict` | The node's `capturedPodUID` is a different Pod. | Compare namespace, owner UID, node ID and Pod UID; this is usually a replaced Pod. Never copy the UID onto the replacement. |
| `identity-conflict` | The Pod's owner, instance ID or node could not be matched; see `detail`. | Inspect the objects named in `detail` and take fresh snapshots. |
| `owner-not-in-snapshot` | The owning Workflow is not in `workflows.json`. | List the namespace again or read the Workflow by name; if the API confirms it is gone, the controller's orphan cleanup applies. |
| `owner-deleting` | The owning Workflow has a `deletionTimestamp`. | Let the existing owner cleanup run; do not delete a Workflow only to get rid of Pods whose history you want to keep. |
| `data-unavailable` | The node status could not be decoded, or the offloaded nodes were not supplied; see `detail`. | Restore access to the API or the referenced storage, or pass `--offload` or `--hydrated-workflows`, then take a fresh snapshot. |

Rows with `legacy-no-receipt` also carry a `legacyClass` hint, for example `multiple-nodes`, `external-outputs`, `memoization-transformation` or `possible-simple-success-controller-verification-required`.
Only the last of these can be captured by the controller itself, and only after the controller has verified it; the hint is a triage aid and does not repeat the controller's checks.

The controller logs why it is holding a Pod in the `captureReason` field:

| `captureReason` | Meaning | What to do |
|---|---|---|
| `result_pending` | The result, or its task result synchronization, has not been saved yet. | Let normal reconciliation continue. |
| `data_unavailable` | An API read, a storage read or a decode failed. | Restore access to the API or to the exact data the Workflow references. |
| `legacy_proof_unavailable` | The saved result is from an older controller and is outside the narrow case the controller can verify. | Keep the evidence and either leave the Pod held or make the explicit decision below. |
| `identity_conflict` | The Pod, its owner or its node does not match what the controller expected. | Inspect the exact objects; the controller retries from fresh reads. |
| `result_conflict` | The result rebuilt from the Pod differs from the saved result. | Keep the saved outcome and investigate the difference. |
| `receipt_not_retained` | The controller wrote `capturedPodUID` but the field was missing when it read the Workflow back. | Check the installed full CRD and look for older components that rewrite node status, then let the controller retry. |

The controller logs a warning the first time it holds a Pod and whenever the reason or detail changes, and logs identical repeated holds at debug level while keeping its retry schedule.
This suppression remembers the 1,024 most recently seen queue keys; after eviction or a restart the warning appears again, and lower-level dependencies can still log their own messages.
Temporary failures are retried without another Pod event, and a controller restart rebuilds the pending cleanup work.
A Pod the controller cannot verify stays held until its evidence or its Workflow changes, or until you release it.
Do not set `capturedPodUID` by hand, and do not change the `completed` label, phase, outputs, task completion state or timestamps to make the controller accept a Pod.

## Save the evidence for one Pod, including offloaded nodes

Set `CAPTURE_POD` and `CAPTURE_WORKFLOW` from the Pod's owner reference, not from a name you assume.
Read the live objects and any task results that still exist:

```sh
kubectl -n "$CAPTURE_NAMESPACE" get pod "$CAPTURE_POD" -o json > capture-evidence/pod.json
kubectl -n "$CAPTURE_NAMESPACE" get workflow "$CAPTURE_WORKFLOW" -o json > capture-evidence/workflow.json
kubectl -n "$CAPTURE_NAMESPACE" get workflowtaskresults \
  -l "workflows.argoproj.io/workflow=$CAPTURE_WORKFLOW" -o json > capture-evidence/task-results.json
```

For uncompressed nodes and the default gzip compression, the helper reads the Workflow snapshot directly.
For offloaded nodes, keep both the Workflow, which holds the reference, and the nodes stored under that reference.
An Argo Server can return the Workflow with its nodes filled in through its normal read API:

```sh
argo get "$CAPTURE_WORKFLOW" -n "$CAPTURE_NAMESPACE" \
  --argo-server "$CAPTURE_ARGO_SERVER" -o json > capture-evidence/hydrated-workflow.json
```

Use your existing authentication and TLS configuration, and keep tokens out of the evidence directory.
Pass that file with `--hydrated-workflows capture-evidence/hydrated-workflow.json`; the helper accepts it only when namespace, name, UID, `resourceVersion` and the rest of the status and spec match the plain Workflow snapshot.
If the versions differ, repeat both reads until they return the same Workflow version; do not edit a `resourceVersion` to force a match.
Going through the Server also covers compression formats other than gzip, which the standalone helper does not decode.

If the Server is unavailable but you have read access to the database, export the one row selected by the Workflow's UID and `offloadNodeStatusVersion` under the configured cluster name and table.
The following PostgreSQL example uses an existing connection service and quoted `psql` variables; take the values from the Workflow and from the persistence configuration:

```sh
psql "service=$CAPTURE_DB_SERVICE" -X -qAt \
  -v offload_table="$CAPTURE_OFFLOAD_TABLE" \
  -v cluster_name="$CAPTURE_CLUSTER_NAME" \
  -v workflow_uid="$CAPTURE_WORKFLOW_UID" \
  -v namespace="$CAPTURE_NAMESPACE" \
  -v node_version="$CAPTURE_NODE_VERSION" > capture-evidence/offload.json <<'SQL'
SELECT json_build_object('namespace', namespace, 'uid', uid,
                         'version', version, 'nodes', nodes::json)
FROM :"offload_table"
WHERE clustername = :'cluster_name' AND uid = :'workflow_uid'
  AND namespace = :'namespace' AND version = :'node_version';
SQL
```

Use the equivalent read-only query for MySQL or MariaDB, keeping the same four JSON keys and an object-valued `nodes` field.
Do not pick the newest row by time, and do not substitute another version of the Workflow.
Pass the file with `--offload capture-evidence/offload.json`; the helper rejects a row with a different namespace, UID or version, more than one matching row, and a row without node data.
A failed query or a missing row does not show that the Workflow is gone.
Do not delete database rows as a recovery step.

Create a private, validated bundle, adding the `--offload` or `--hydrated-workflows` option you used above if the nodes are compressed or offloaded:

```sh
python3 hack/status-capture.py export \
  --pod capture-evidence/pod.json \
  --workflow capture-evidence/workflow.json \
  --task-results capture-evidence/task-results.json \
  --output-dir capture-evidence/retained-pod
```

The helper creates a new directory with mode `0700` containing the original objects, the decoded nodes and a SHA-256 manifest, all with mode `0600`.
It refuses to overwrite an existing bundle.
Save the Pod logs separately if you need them and they are still available.
Exporting this evidence does not record the result in the Workflow status, and it cannot restore output that the executor never saved.

## Remove a finalizer yourself

For a Pod the controller cannot verify, the default is to leave the finalizer in place and restore the missing evidence.
Deleting the Workflow is not the only way out.
After saving and reviewing the evidence, you can remove the finalizer from one specific finished Pod while leaving the completed Workflow untouched.
Doing so accepts that the controller never verified this Pod's result, and once the Pod is gone any details or outputs that were never saved are lost for good.
It is a manual decision, not something the controller does on its own, and it does not mean the old result has been reconstructed.

For an ordinary result from an older controller, the helper prepares this patch only when the Workflow is completed and still exists, the Pod's node is terminal and has no `capturedPodUID`, the node data is available, and no task result is known to be unsynchronized.
It refuses a running Workflow, mismatched identities, a `capturedPodUID` that names another Pod, missing offload data and a Pod whose result the controller has already verified.
A task result whose completion state was never recorded is reported as such, and you have to weigh that missing evidence in the decision.
Do not do this while anything else is retrying or changing the Workflow; note the completed Workflow version you reviewed and read both objects again before preparing the patch.

A Workflow that ended with the oversize storage `Error` and no offloading has its own inventory reason, `completed-storage-error-without-capture`.
Its last saved node and task state can be incomplete even though execution finished.
Save the Pod and the available `WorkflowTaskResult` data before releasing anything; the Workflow keeps its `Error` and its incomplete node history.
For this case only, add `--acknowledge-incomplete-result` to the command below in addition to `--acknowledge-unproven-capture`.
The extra flag accepts the loss of the unsaved result; it does not add a `capturedPodUID`, reconstruct outputs or change the Workflow.
The helper still refuses a Pod with a `capturedPodUID`, a pending restart, mismatched identities, and a Workflow that did not end with this specific size and offload error.

Once you have reviewed the bundle and accept the loss, prepare the patch from fresh snapshots:

```sh
python3 hack/status-capture.py prepare-release \
  --pod capture-evidence/pod.json \
  --workflow capture-evidence/workflow.json \
  --acknowledge-unproven-capture \
  --output capture-evidence/release-patch.json
```

Add the same `--offload` or `--hydrated-workflows` option you used for the export if the nodes are compressed or offloaded.
Review the JSON, then apply it in a separate step:

```sh
kubectl -n "$CAPTURE_NAMESPACE" patch pod "$CAPTURE_POD" \
  --type=json --patch-file capture-evidence/release-patch.json
```

The patch first tests the exact Pod UID, `resourceVersion` and finalizer list, then removes only `workflows.argoproj.io/status` at the index it found.
If the Pod's metadata changed in the meantime, or a Pod with the same name replaced it, the patch fails; take fresh snapshots and review again rather than removing the tests.
The helper never applies the patch, never writes `capturedPodUID` and never changes Workflow status.
The Pod preconditions do not lock the Workflow, so check the completed Workflow version separately.

Finally, read the Workflow and the Pod again, or record the NotFound response if the Pod was deleted.
Confirm that the Workflow's phase, message, outputs, node data and completion timestamps are unchanged; only API metadata should have moved.
Confirm that the intended Pod lost only Argo's finalizer and that any finalizers from other systems are still there.
If another finalizer keeps the Pod from being deleted, its owner is responsible for that part.
Keep the evidence, the reviewed patch and the record of what you did together, as the record of this decision.
