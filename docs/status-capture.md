# Pod Status Capture and Retained Pods

When `ARGO_POD_STATUS_CAPTURE_FINALIZER=true`, Argo keeps `workflows.argoproj.io/status` on a terminal Pod until the corresponding node result and applicable task-result synchronization are persisted.
The optional node field `capturedPodUID` binds that result to the observed Pod incarnation.
For offloaded nodes, the stored Workflow must reference the saved version; a database write alone is insufficient.
A confirmed deleting, absent or replaced owner has a separate cleanup rule, which does not assert result capture.

## Upgrade and rollback

Install the new full Workflow CRD before starting the new controller.
Use matching controller, Argo Server and CLI versions as required by the [supported version policy](releases.md#supported-version-skew).
Keep the database configuration, controller instance ID and namespace scope consistent across the transition.
A successful controller rollout alone does not establish that the API retained the new field.
Inspect a newly completed Workflow and its exact Pod UID, hydrating node data when necessary.

The following controlled transition preserves outstanding capture obligations by excluding old active controllers and writers.
It is not a requirement to stop every installation with status capture disabled, and it does not replace the general [HA rollout guidance](high-availability.md#deployment-rollout-strategy).
An ordinary rolling upgrade must not be described as continuously providing the new guarantee while an old binary can still become active.
Stopping the controller does not stop workloads that are already running.

Use this order when existing Pods need the new protection:

1. Pause new submissions and relevant CronWorkflows using the installation's existing controls.
   Quiesce operations that rewrite existing Workflows, including resume, suspend, retry and selected-node updates, while aligning the Server and direct Kubernetes CLI versions.
2. Record the old controller image, full CRD, instance scope and persistence configuration without exporting credentials.
   Inventory retained Pods and preserve the data for existing cleanup obligations.
3. Stop all old controller replicas for that scope and verify that their processes have exited.
   An old standby does not process cleanup while it remains standby, but can resume old cleanup behavior if it becomes leader.
4. Apply the full CRDs from the same release as the new controller, then start only new controller replicas with status capture enabled.
5. Verify the elected leader's image, Workflow and node outcomes, actual retained `capturedPodUID` values and cleanup of the matching Pod UIDs.
   Classify retained legacy Pods using the procedure below, then resume submissions.

The new guarantee starts when the compatible new controller is active with a schema that preserves its receipt, the required storage is accessible and old writers cannot rewrite the affected results.
An old controller does not acquire the new cleanup check from a CRD update.
Do not promise uninterrupted new protection during a mixed rollout that allows an old controller to become active.

For rollback, pause submissions and let the new controller drain active Workflows and supported cleanup obligations first.
Inspect every retained Pod; resolve temporary failures, retain unsupported legacy evidence, or make a separate explicit operator disposition as described below.
Do not start an old controller while relying on it to protect outstanding new capture obligations.
If those obligations cannot be drained, keep the new controller or leave the affected scope stopped while preserving its Pods and data.
Keep the newer CRD during rollback; reverting the schema can discard new status fields.
Older typed writers can lose fields they do not know when they read and rewrite node data, so do not treat optional-field decoding as receipt preservation.
Argo Server resume, retry and selected-node updates can rewrite hydrated node data; a CLI using the direct Kubernetes transport can also perform typed updates.
Whole-Workflow stop and terminate use a narrow patch and do not by themselves rewrite node status.
For commands routed through Argo Server, the Server performs these node writes; the CLI version alone does not establish the serialization behavior.
The new protection ends when an old controller starts processing that scope.
To return, stop the old replicas, verify the current full CRD, restart the new controller and repeat the inventory and receipt checks.
This procedure does not restore already deleted Pods or receipts already removed by another writer.

## Completed Workflows from older controllers

A new receipt can be recorded for a deliberately limited legacy case: a completed, single-node successful Workflow running an ordinary container, with a matching terminal successful Pod and stable execution template.
The controller requires complete real container statuses, explicit applicable task synchronization and equality of the supported result, including exit code, message, finish time, progress, host and resource duration.
The stored template must agree with the template carried in the Pod's `ARGO_TEMPLATE`; definitions that were offloaded into an environment source are outside this path.
Only the new UID association is added; the completed outcome and execution timestamps remain unchanged.
The controller then reads the actual persisted result before allowing its existing guarded cleanup.

This path excludes memoization, daemon or stop dispositions, deadlines, retries and automatic restarts, multiple-node graphs, inputs, external outputs and unsupported execution metadata.
An ordinary successful two-step Workflow is outside this path because its status contains multiple nodes.
These exclusions describe the current verifier's support boundary, not a claim that every excluded result is impossible to reconstruct.
User sidecars or init containers, template-level Pod specification patches, hooks, external template references and nonzero container restart counts are also outside this narrow class.
A successful Pod can legitimately have a saved node Error because a later memoization save failed.
Even if the cache is now healthy, the controller must retain that Error and must not repeat the save or replace the outcome with Succeeded.
A WorkflowTaskResult addresses a node and owning Workflow, and is not proof of the Pod UID that produced its data.
Active legacy Workflows still use normal reconciliation; existing receipts retain the existing nil and legacy task-completion semantics.

## Find and classify retained Pods

Use a private local directory and the normal credentials for the affected namespace.
Snapshots can contain sensitive input values, environment values, artifact locations and output data; keep them private and exclude kubeconfig files and credentials from shared evidence.
Set `CAPTURE_NAMESPACE` and run the helper from an Argo checkout containing this change.

```sh
umask 077
mkdir capture-evidence
kubectl -n "$CAPTURE_NAMESPACE" get pods -l workflows.argoproj.io/workflow -o json > capture-evidence/pods.json
kubectl -n "$CAPTURE_NAMESPACE" get workflows -o json > capture-evidence/workflows.json
python3 hack/status-capture.py inventory \
  --pods capture-evidence/pods.json \
  --workflows capture-evidence/workflows.json
```

The helper uses only saved JSON files and never contacts the API or database.
Its offline regression tests run with `make test-status-capture` from a checkout, using only the Python standard library.
Its output is an observation of those files, not an authorization to remove a finalizer.
In particular, an owner missing from a list snapshot is not proof of an authoritative API NotFound.
Controller logs supply current hold reasons; the inventory groups evidence using these states:

| Observation | Next action |
|---|---|
| Matching receipt in snapshot | Re-read the current API and applicable task state; allow normal cleanup, policy delay and retries. |
| Active Workflow or pending result/task state | Restore normal reconciliation and output dependencies; do not force completion. |
| Data unavailable | Restore API or referenced storage access, then collect a fresh snapshot. |
| Legacy result without receipt | Check whether the controller's limited same-result capture applies; otherwise retain evidence or choose explicit operator disposition. |
| Identity conflict | Compare namespace, Workflow owner UID, node identity and Pod UID; do not copy a receipt to a replacement. |
| Persisted restart disposition | Allow retirement of the exact `restartingPodUID` through the controller. |
| Owner deleting | Allow the existing verified owner cleanup rule; do not use owner deletion solely to discard a history you want to retain. |

The controller's `captureReason` field distinguishes the actual hold:

| Log reason | Meaning and continuation |
|---|---|
| `result_pending` | The result or applicable task synchronization has not been persisted; let normal reconciliation continue. |
| `data_unavailable` | An API, storage or decoding operation failed; restore access or the exact referenced data. |
| `legacy_proof_unavailable` | The completed legacy state is outside supported fresh capture; preserve evidence and retain it or make an explicit operator decision. |
| `identity_conflict` | Pod, owner or node identity contradicts the proposed capture; inspect the exact objects and retry only from fresh evidence. |
| `result_conflict` | Fresh interpretation differs from the saved result; keep the completed outcome and investigate the difference. |
| `receipt_not_retained` | A write did not retain the new receipt on authoritative read; check the installed full CRD and incompatible writers before retrying. |

The inventory's legacy subtype is a triage hint, not a second implementation of the controller's verifier.
Do not fill `capturedPodUID` manually or change `completed`, phase, outputs, task-completion state or timestamps to force acceptance.
Temporary failures are retried without requiring another Pod event; a controller restart also reconstructs cleanup work.
The controller warns on the first hold or a changed reason/detail and records identical repeated holds at debug level, while retaining its retry schedule.
This suppression covers the 1,024 most recently observed queue keys; eviction or process restart can produce another warning, and underlying dependency code can still log its own messages.
An unsupported legacy result remains held until its evidence or owner disposition changes, or an operator deliberately releases it.

## Preserve exact evidence, including offloaded nodes

For one retained Pod, set `CAPTURE_POD` and `CAPTURE_WORKFLOW` from the verified owner reference rather than a name guess.
Read the live objects and any remaining task results:

```sh
kubectl -n "$CAPTURE_NAMESPACE" get pod "$CAPTURE_POD" -o json > capture-evidence/pod.json
kubectl -n "$CAPTURE_NAMESPACE" get workflow "$CAPTURE_WORKFLOW" -o json > capture-evidence/workflow.json
kubectl -n "$CAPTURE_NAMESPACE" get workflowtaskresults \
  -l "workflows.argoproj.io/workflow=$CAPTURE_WORKFLOW" -o json > capture-evidence/task-results.json
```

For raw nodes or default gzip compression, the helper validates and reads the supplied Workflow directly.
For offload, preserve both the original Workflow reference and the actual nodes at that reference.
A configured Argo Server can hydrate the Workflow through its normal read API:

```sh
argo get "$CAPTURE_WORKFLOW" -n "$CAPTURE_NAMESPACE" \
  --argo-server "$CAPTURE_ARGO_SERVER" -o json > capture-evidence/hydrated-workflow.json
```

Use the installation's existing authentication and TLS configuration; do not place tokens in the evidence bundle.
The helper accepts `--hydrated-workflows capture-evidence/hydrated-workflow.json` only when namespace, name, UID, resourceVersion and non-node status/spec agree with the original Workflow snapshot.
If versions differ, repeat the two reads until they identify the same stable Workflow version; do not edit their resourceVersions to force a match.
This route also supports non-gzip compression through the server's existing decompression support.
The standalone helper does not add Python compression dependencies.

If the server is unavailable but authorized database read access exists, export the actual row selected by the Workflow's UID and `offloadNodeStatusVersion` under the installation's configured cluster name and table.
The following PostgreSQL example uses an existing connection service and quoted `psql` variables; supply the exact values from the Workflow and persistence configuration:

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

Use the equivalent read-only query for MySQL or MariaDB, preserving the same four JSON keys and an object-valued `nodes` field.
Do not select the newest row by time or substitute another Workflow version.
Pass `--offload capture-evidence/offload.json` to the helper; it rejects a different namespace, UID or reference, duplicate matches and unavailable node data.
SQL access failure or a missing row is not evidence that the Workflow owner is absent.
Do not delete storage rows as a recovery step.

Create a private validated bundle, adding the matching `--offload` or `--hydrated-workflows` option when required:

```sh
python3 hack/status-capture.py export \
  --pod capture-evidence/pod.json \
  --workflow capture-evidence/workflow.json \
  --task-results capture-evidence/task-results.json \
  --output-dir capture-evidence/retained-pod
```

The helper creates a new directory with mode `0700`, files with mode `0600`, the original objects, decoded node data and a SHA-256 manifest.
It refuses to overwrite an existing bundle.
Preserve relevant Pod logs separately if they are needed and still available.
Exporting this evidence does not mean Argo has captured the Pod result in Workflow status, and does not restore output the executor never saved.

## Explicit operator disposition while retaining Workflow history

Keeping the finalizer and restoring missing evidence remains the default for an unresolved legacy result.
Deleting the Workflow is not the only possible choice.
An operator may deliberately retire a specific terminal Pod after preserving and reviewing the available evidence, while leaving the completed Workflow unchanged.
This waives the missing automatic capture proof and may permanently lose remaining Pod details or unsaved outputs once the Pod disappears.
It is not an automatic fallback or a claim that the old result has been reconstructed.

The helper prepares this action only for a matching live completed owner, a terminal Pod node with no receipt, available node data and no known incomplete task synchronization.
It rejects an active Workflow, conflicting identities, a different nonempty receipt, missing offload data and already proven receipts.
Legacy unspecified task state is reported explicitly; the operator must account for that missing evidence in the decision.
Do not make this decision while another actor is retrying or changing the Workflow; preserve the completed version under review and re-read both objects before preparation.

After reviewing the private bundle and accepting that limitation, prepare a patch from fresh snapshots:

```sh
python3 hack/status-capture.py prepare-release \
  --pod capture-evidence/pod.json \
  --workflow capture-evidence/workflow.json \
  --acknowledge-unproven-capture \
  --output capture-evidence/release-patch.json
```

Add the same matching storage option used for export when nodes are encoded or offloaded.
Review the JSON before executing the separate API operation:

```sh
kubectl -n "$CAPTURE_NAMESPACE" patch pod "$CAPTURE_POD" \
  --type=json --patch-file capture-evidence/release-patch.json
```

The patch tests the exact Pod UID, resourceVersion and original finalizer list, then removes only `workflows.argoproj.io/status` at its verified index.
A concurrent metadata change or same-name replacement causes the patch to fail; collect fresh data and review again instead of weakening the tests.
The helper never applies the patch, writes `capturedPodUID` or modifies Workflow status.
There is no cross-object transaction: the Pod preconditions do not lock the Workflow, so preserve and verify the completed Workflow version separately.

Finally re-read the Workflow and Pod, or record the Pod's actual API NotFound if it was deleted.
Verify that Workflow phase, message, outputs, all node semantics and completion timestamps remain unchanged; only API metadata may have advanced independently.
Verify that the intended Pod UID lost only Argo's finalizer and that any foreign finalizers remain.
If a foreign finalizer prevents physical deletion, its owner is responsible for that remaining work.
Keep the evidence, the reviewed patch and the action/verification record together as an explicit operator decision.
