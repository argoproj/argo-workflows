<!-- cspell:words codegen -->
# Local parameterization experiment

This is an opt-in experimental implementation with explicit supported boundaries in [the implementation decisions](implementation-decisions.md).
The local compatibility checkout is pinned to v4.0.1; the community contribution branches target upstream main and release-4.0.
Local acceptance does not imply upstream design approval or a passing upstream CI run.
No delivery runtime/API/dashboard/MCP source is part of this change.

## Isolation

The development installation uses a separate local k3d/k3s cluster:

- Cluster/context: `k3d-argo-parameterization`
- Kubernetes: `rancher/k3s:v1.32.8-k3s1`
- Namespace: `argo-parameterization-lab`
- Argo controller instance: `parameterization-lab`
- UI via local port forward: `http://127.0.0.1:2747/parameters/argo-parameterization-lab`

The pre-existing Rancher Desktop Kubernetes cluster reached its Pod limit. The
experiment was moved to the separate cluster, preserving pre-existing workloads.
The Docker VM inotify instance limit was increased from 128 to 8192 to support the
additional local control plane. No remote cluster uses these manifests/images.

The lab uses Argo's server authentication mode over a loopback port forward.
Do not expose this lab service publicly. Real installations must configure Argo
client/SSO authentication and the caller's Kubernetes RBAC.

![Structured parameter editor with synthetic demonstration values](images/structured-parameters.png)

## Named definition versions

Open a saved definition in **Resource definitions and records** and choose **Publish named version**.
Confirm a distinct name and optional note to create an immutable object of the same definition kind.
The source is unchanged and no Workflow is submitted.
**View published versions** loads one family page; open or compare a version on demand.
Restore with **Clone as new definition**, then explicitly select that name in a binding before the normal save, preview and submit flow.
The API-server immutable-snapshot policy also blocks direct spec or provenance edits on published objects.
Versions remain readable after source deletion; retrying the same publication request recovers the same UID.

This versions a single definition, not an atomic release of all transitive dependencies.
Authors can compose published names; execution submission still freezes the full reference closure separately.
Run `test/parameterization/definition_publications_e2e.py` for the local browser path across all six definition kinds, admission protection, historical restore and a real native Resource result.

## Paged revision comparisons

Saved-revision comparisons return at most 100 changes by default (1–200 can be
requested). Value, model and source changes share one stable page order, with
per-layer totals and a continuation position bound to the exact comparison.
Every page rereads the immutable pair under current permissions and validates
sensitive references. A cursor is not an access token; a changed pair, withdrawn
permission or malformed cursor is rejected.

The parameter editor provides Previous/Next diff page controls and only renders
the current page. Its path filter explicitly applies to that page. Empty sections
with changes on another page are not described as globally unchanged. Restoring
a revision remains a separate edit/save action.

Run `python3 test/parameterization/paged_diff_e2e.py` in the isolated lab for
save → 260-field comparison over three pages → restore → native Resource action.
The existing paged-documents, structured-diff and execution-diff cases remain
regressions. Long-history pagination uses the verified continuation described below.

## Paged execution comparisons

The saved Workflow view uses the same bounded Diff navigation across parameter,
model, source, binding, form, action, implementation and compiled-input layers.
`execution-diff` defaults to 100 total changes per page, with a maximum of 200.
Every request rechecks the caller's access to source receipts and target definitions.
The page fingerprint excludes status-only Workflow resourceVersion changes.
An input, binding, form or implementation change invalidates the comparison;
restart comparison rather than mixing changes from different snapshots.
Returning to the first page also submits the original fingerprint.
Missing historical definitions remain explicitly unavailable on every page.
Comparing never applies parameters, retries or starts a Workflow.

Run `python3 test/parameterization/paged_execution_diff_e2e.py` in the isolated lab.
It freezes a Resource Workflow, changes 130 values and 130 form labels, traverses
all Diff layers, rejects a definition change between pages, then resumes the
original Workflow and checks that its original ConfigMap result is unchanged.

## Long parameter histories

History pages no longer scan again from the latest revision or stop at 1000 revisions.
The server signs a continuation position only after traversing the committed
parent chain from the ParameterSet head.
Each page rereads the set, original anchor and returned revisions with the
current caller's Kubernetes client; the cursor grants no additional access.
A concurrent save appears in a new query, without inserting or duplicating rows
in the history query already in progress.
Uncommitted revisions from failed compare-and-swap saves remain excluded.
No unbounded history array is added to a CR, and no second workflow engine or
persistent cursor database is introduced.

The cursor is namespaced and bound to the ParameterSet UID and immutable anchor
UID/resourceVersion, with a renewable 15-minute expiry.
For multiple Argo Server replicas, mount the same 32 random bytes from a Secret
and set `ARGO_PARAMETERIZATION_HISTORY_KEY_FILE` to that read-only file.
The single-server lab can use the generated process key.
Restart or key rotation then returns HTTP 410; click **Reload history**, without
saving parameters or creating a Workflow.
A malformed cyclic chain is rejected; a query is explicitly bounded at one
million revisions rather than silently truncated.
This supports sequential history browsing, not random page-number access.

`test/parameterization/long_history_e2e.py` saves 1006 real local parameter
revisions, excludes an orphan, changes the head during pagination, restores the
first version and executes the restored value through a native Resource action.

## Reproduce

Use a dedicated kubeconfig and explicit context. Do not merge it into the default
kubeconfig or run these commands against an existing business cluster.

```sh
k3d cluster create argo-parameterization --image rancher/k3s:v1.32.8-k3s1 \
  --api-port 127.0.0.1:6551 --kubeconfig-update-default=false \
  --kubeconfig-switch-context=false \
  --k3s-arg '--disable=traefik,servicelb,metrics-server@server:0' \
  --k3s-arg '--secrets-encryption@server:0' \
  --k3s-arg '--secrets-encryption-provider=secretbox@server:0'
k3d kubeconfig get argo-parameterization > /private/local-k3d.kubeconfig
chmod 600 /private/local-k3d.kubeconfig
export KUBECONFIG=/private/local-k3d.kubeconfig
```

Build the UI, `dist/argo` with the `ui` tag, and `dist/workflow-controller` for
Linux amd64 using the repository's supported Go toolchain. Build the local image
with `manifests/parameterization/Dockerfile.local`, then import it with
`k3d image import IMAGE -c argo-parameterization`.
These steps are sequential: await the UI build, then the embedded Argo binary,
then the image build, then import and rollout. Use a new local tag for each
changed binary. `ARGO_CHECK_LOCAL_ASSETS=true` checks the served UI bundle against
the completed local build before the E2E creates any Workflow.

Install the native CRDs from this checkout in this new cluster only, then the experimental CRDs and
namespace manifests. Wait for CRD establishment before applying the demos:

```sh
kubectl --context k3d-argo-parameterization apply --server-side -k manifests/base/crds/full
kubectl --context k3d-argo-parameterization apply -f manifests/parameterization/immutable-snapshots.yaml
kubectl --context k3d-argo-parameterization apply -f manifests/parameterization/crds.yaml
kubectl --context k3d-argo-parameterization wait crd --all --for=condition=Established --timeout=60s
kubectl --context k3d-argo-parameterization apply -f manifests/parameterization/local-install.yaml
kubectl --context k3d-argo-parameterization apply -f examples/parameterization/demo.yaml
kubectl --context k3d-argo-parameterization -n argo-parameterization-lab \
  port-forward svc/argo-server 2747:2746
```

The image in the Deployment must match the imported local tag. Image pull policy
is `Never`; there is no registry push or remote deployment.

## User walkthrough

1. Open the local UI. Dismiss the upstream first-run survey/release notes if shown.
   Alternatively open a native WorkflowTemplate such as `training-demo`. Its
   details and Submit panel discover the namespaced bindings; choose `training`
   to open its exact model in the structured editor. Multiple bindings are
   listed separately. Templates without bindings retain their native entry.
2. Select `dynamic-delivery`, enter a new parameter set name, switch to YAML and
   paste `examples/parameterization/topology-values.yaml`.
3. Switch back to Form. Expand zones, subnets and roles. Add/remove/reorder items.
   Counts drive native Argo loops; no business account or real cloud is involved.
4. Save. Change a value and save again. Select Compare from/to in history.
   Restore a prior revision into the editor, then save to create a new revision.
5. Validate and preview, inspect the native Workflow and submit explicitly.
6. Open the created native Workflow. Resume the approval suspend node. The demo
   creates one namespace-local ConfigMap per zone/subnet/role combination.
7. View the original parameter snapshot from Workflow details and compare it with
   the latest draft. “Edit a parameter version” loads that execution's binding,
   set and revision; the current draft supplies its CAS version and a comparison.
   Save edits, preview and explicitly submit as a **new Workflow**. The new
   submission records the source Workflow name and UID; the original execution
   remains unchanged. Editing uses the current binding/form and rejects a changed
   parameter model identity. Missing/deleted source resources fail explicitly.
8. The `training` binding uses another schema and a native container action.
9. Select `secret-demo`: enter a **synthetic** value beginning `synthetic-` (at least
   16 characters), store it with the encrypted-input widget, then save/preview/submit.
   YAML/history expose only a Secret reference. The container reports
   `credential-consumed` without printing the value.
10. Select `intermediate-demo`, save the empty initial set and submit. In Workflow
    details, the native suspend presents `release` and `replicas`. Submit those
    inputs once. The downstream container consumes the supplied outputs; a receipt
    remains visible after the node completes.

For an intentional failure, set `fail: true` and submit another revision. The
final check fails after resource tasks succeed. Create a ConfigMap named
`<workflow-name>-recovery` with `data.allow: "true"` in the lab namespace, then use
Retry Workflow with the original revision. The Workflow UID and successful node
results should be retained. This simulates recovery of an external dependency.

Changed-parameter partial retry supports a completed, frozen DAG with explicit dependencies and statically named nested DAG calls.
From an execution, save a revision, choose **Preview partial retry**, inspect affected tasks and retained results, then explicitly apply it to the same Workflow.
Retained leaf results require `parameters.workflows.example.io/result-reuse: same-inputs`; rerun non-suspend tasks require an implemented `parameters.workflows.example.io/retry-safe: idempotent` contract.
The server validates native Argo retry preserves unrelated concrete results and approvals, and freezes the original execution in a ParameterRetry receipt.
Template graph edits, artifact reuse in new-parameter retry, exit hooks and unsupported execution shapes are rejected.
Author-approved whole-loop replacement is covered by the additional topology suite below.
A separate new submission remains explicitly a new Workflow.
Run `python3 test/parameterization/partial_retry_e2e.py` against the isolated lab to exercise save, fail, edit, diff, impact preview and native same-UID retry.

## Bounded automated checks

```sh
go test ./util/parameterization ./server/parameterization ./server/workflow ./workflow/util
(cd ui && yarn test src/parameterization --runInBand && yarn tsc --noEmit)
ARGO_PARAMETERIZATION_URL=http://127.0.0.1:2747 \
ARGO_PARAMETERIZATION_CONTEXT=k3d-argo-parameterization \
ARGO_CHECK_LOCAL_ASSETS=true \
python3 test/parameterization/e2e.py
python3 test/parameterization/schema_e2e.py
python3 test/parameterization/secrets_e2e.py
python3 test/parameterization/inputs_e2e.py
```

The E2E uses Argo Server HTTP APIs, Kubernetes CR persistence, the native
controller/executor, real namespace-local Resource actions, and a second-domain
container workflow. Each wait has a deadline. No cloud credentials or delivery
API connections are used. Test records are retained for inspection.

## Remaining work and supported limits

Latest-at-submit secret resolution, native generated CR clients, action parameter/artifact mappings, Vault KV-v2 materialization and approved loop-expansion changes have complete local-path evidence.
Native leaf retry groups also passed on both upstream main and v4.0.1.
These entries are completed subsets, not work waiting to be implemented again.

The remaining delivery work is per-attempt dynamic credential lifecycle integration, final public API/CLI contract review, full community checks, the release-branch adaptation and the two public PRs.
The native REST extension currently shares the resolver and admission contracts; it is not yet a generated gRPC service or a new CLI command family.

New-parameter partial retry deliberately rejects arbitrary frozen template replacement, per-item reuse across changed loop identities, memoization, composite retry groups and unsupported artifact forwarding/storage.
Those cases need additional semantic proofs; the UI/API reports unsupported rather than recreating an entire Workflow or fabricating successful results.
The implemented HTTP checksum subset is described below.
Production-specific provider adapters are outside this generic first implementation; the single-operation reference adapter exercises real local asynchronous recovery and cancellation.

The separate historical delivery backlog remains in the calling workspace and cannot be closed by local Argo checks.
Community review is pending; local acceptance does not imply upstream acceptance.

## Security backend setup

`ARGO_PARAMETERIZATION_SECRET_BACKEND=kubernetes-encrypted` is an administrator
assertion that the Kubernetes API server encrypts Secrets at rest and does not
record Secret bodies in audit logs. Do not enable it on an unencrypted cluster.
`ARGO_PARAMETERIZATION_ALLOW_LOOPBACK_SECRET_INPUT=true` is local-lab only.
The API rejects non-TLS secret writes from non-loopback peers and does not trust
caller-provided forwarded-protocol headers. Synthetic test key rotation is opt-in
with `ARGO_TEST_SECRET_ROTATION=true` and targets the named dedicated k3s only.
The rotation sequence follows [K3s documentation](https://docs.k3s.io/security/secrets-encryption).

## Resource definitions and records

Open **Structured parameters → Resource definitions and records**. Select ParameterTemplate, WorkflowForm, WorkflowParameterBinding, ActionDefinition, `ActionResultTemplate`, or WorkflowActionBinding to inspect the actual namespaced resources and their YAML references. Saved sets, immutable revisions, submissions, retries, and intermediate inputs are also listed. Lists use Kubernetes pagination and the existing caller permissions. This first catalog is read-only; definition creation and updates still use the included Kubernetes manifests.

## Typed CR API and schema upgrade

The API types live in `pkg/apis/parameterization/v1alpha1`: registered Kind/List
pairs, typed spec/status, generated DeepCopy and generated structural CRDs.
`hack/parameterization-codegen.sh` uses controller-gen v0.18.0, matching the native
repository generator. Open JSON is limited to business values/schema documents
and embedded native Argo implementations. Native validation remains responsible
for the latter; an embedded object marker alone does not validate a Workflow.

This experiment requires Kubernetes >=1.30 and the accompanying immutable
snapshot admission policy. CRD CEL cannot inspect unknown-type JSON; the policy
compares the actual JSON including nested values and permits no-op/metadata-only
updates. Install the policy before upgrading the structural CRDs. The schema E2E
checks allowed no-op updates as well as rejected mutations, so an always-reject
rule cannot produce a false pass. Native status subresources enforce a separate
RBAC boundary. Historical inputs that predate `submittedAt` remain readable.

ResourceOperation has a typed optional contract and a reference external operation
controller in this installation.
The optional definition status controller is described below; the supported form layout and interaction subset follows. This is not full admission parity
for business model rules or unimplemented optional extensions.

## Frozen nested template references (v12)

Install `examples/parameterization/nested-refs.yaml` and select `nested-library-demo`.
Static DAG/steps/hook references, referenced local templates, inline templates and
recursive calls are frozen into native templates. The caller's Kubernetes client
resolves each namespaced or cluster-scoped resource once. Preview/submit checks the
complete referenced spec digest. Native dependencies, loops, outputs and scheduling
remain unchanged. The original execution and original-parameter retry do not resolve
new source definitions. Dynamic reference names are explicitly rejected.

Limits: 64 referenced resources, 256 external templates and 64 nesting levels.
Top-level bindings still target namespaced WorkflowTemplates; cross-namespace access
is not introduced. The local lab grants its server GET on ClusterWorkflowTemplates
through a dedicated read-only ClusterRole; production authentication must use caller RBAC.

`test/parameterization/template_refs_e2e.py` tests nested DAG/steps/cluster references,
preview invalidation, deletion after submission, actual loop output propagation and
same-UID retry with preserved successful nodes. It removes only per-run source fixtures.

## Cluster-scoped root templates

Install `examples/parameterization/cluster-template.yaml` after `demo.yaml` in the isolated lab.
`WorkflowParameterBinding.spec.clusterScope: true` resolves `workflowTemplate` as a native ClusterWorkflowTemplate using caller RBAC.
The parameter resources and resulting Workflow remain namespaced.
An action binding must target the same name and scope.
The native ClusterWorkflowTemplate details and Submit panel link to compatible bindings in the chosen namespace.
Preview freezes the root and nested template closure; source changes invalidate stale previews, while committed executions and original retries retain their snapshot.
Run the nested-template full-path E2E with `ARGO_PARAMETERIZATION_CLUSTER_ROOT=true` to cover this path.

## Native action contract validation

ActionDefinition schemas are frozen as inert template annotations with each native implementation.
The enhanced Controller validates resolved inputs before dispatch and synchronized Pod/HTTP results before successful downstream execution.
Plain string fields retain native string encoding; fields with a concrete integer/number/boolean/object/array/null type use JSON encoding.
HTTP/script `outputs.result` maps to an explicitly declared `result` output property.
Unbound templates retain native behavior, and the enhanced Server and Controller must be installed together.
Unsupported artifact/plugin/memoization/composite action combinations are rejected during preview.
Run `python3 test/parameterization/action_contract_e2e.py` for real container and namespace-local HTTP success/failure paths.
External-operation recovery and cancellation are exercised through the optional reference controller in `examples/parameterization/resource-operation`; this does not confer idempotence on ordinary native actions.

### Optional admission and mixed retry acceptance

`test/parameterization/install_admission.py` installs a namespace-scoped validating webhook in the dedicated local cluster.
It shares validation with the Argo API, verifies actual caller access before every dependency read, and cannot write Workflows.
Unannotated native workflows are excluded. State updates and native suspension/termination preserve frozen contracts without resolving latest definitions.
Execution input changes require a matching checked ParameterRetry; raw parameter patches are rejected.
Run `admission_e2e.py` for API and direct Kubernetes submission, and `partial_retry_e2e.py` for new-parameter retry followed by original-parameter recovery.

### Revision comparison

The revision UI groups value, model and source changes and filters by JSON Pointer or item identity.
Changed array identity rules trigger whole-array comparison; missing, null, false and zero remain distinct.
`diff_e2e.py` creates three revisions and changes the model, then verifies all comparisons without creating any Workflow.

### External operation demo

Open the `operation-demo-report` binding, choose a fresh set name and save `mode: normal`, `delay: "2"`.
Preview and submit; a native Resource task creates a persistent operation and the optional controller observes the synthetic HTTP service.
`operation_e2e.py` covers lost responses, restart, native same-Workflow retry, cancellation vs detach, unknown outcomes, invalid outputs and forged invocation identity.
The reference controller validates a single native Resource invocation; general nested/loop identities and production providers remain separate work.

### Layered execution comparison

`POST /api/v1/parameterization/{namespace}/execution-diff` accepts
`{"source":{"workflow":"name","uid":"uid"},"revision":"saved-revision"}`.
It uses caller-scoped reads and never submits or retries. The response separates
parameter values/model/sources, parameter and action bindings, form, action
contracts, implementations/workflow structure, and compiled native inputs.
Unresolved task-output expressions remain expressions.

New submission and retry receipts carry typed, immutable `definitions` snapshots.
Original-input retries inherit the adopted snapshot. Older receipts explicitly
report unavailable historical definition layers; current definitions never stand
in for missing history. Their frozen native Workflow remains comparable.
Checked retry plans include `executionDigest` to detect definition drift after
preview, while older receipts without that field remain compatible.

Run `test/parameterization/execution_diff_e2e.py` and `partial_retry_e2e.py` against
the dedicated local lab to cover both frozen-source deletion and mixed new/original
retry. These tests do not contact remote delivery environments.

### Paged document reads

Resource catalog UI requests `view=metadata` (native Kubernetes limit/continue with the caller's transport). It fetches the exact body only when selected; annotations and managed fields are excluded from summaries. Revision history uses `view=summary`; restore fetches the immutable body separately and never submits an execution. `paged_documents_e2e.py` exercises 52 definitions, 27 large input versions, concurrent saves, uncommitted revision exclusion, restore, Diff and native execution.

Long-history continuation and saved/execution Diff paging are implemented as described above; catalog metadata paging is a separate path.

## Declared form layouts and interactions

WorkflowForm layout supports nested group, tab and tree nodes, with fields assigned by group ID.
Field widgets support text, number, checkbox, select, object, array, object-table, YAML and declared sensitive references.
Field presentation policies inherit through ancestors; hidden values remain in the shared parameter document.
Array wildcard field paths bind predicates to the current row, while stable data identities remain a model concern.
Tabs stay mounted so invalid YAML edits and save validation survive tab switches.
Named option sources are mapped to field paths; server preview checks the currently selected values again.
The server rejects invalid layout graphs, unknown fields, incompatible widgets and mismatched model references.
Presentation restrictions are not authorization; v37 adds reference-only sensitive input and v39 adds frozen intermediate option providers, while plaintext outputs remain rejected.

The Python Playwright case `test/parameterization/form_ui_e2e.py` drives a real Chromium browser against the isolated lab.
It saves dynamic form values, previews and submits a native Resource Workflow, reads its ConfigMap result, compares revisions and tests invalid definitions/options.
A second workflow freezes an intermediate form, changes its source, accepts browser input at the native suspend node and verifies the actual downstream container output.
Set `ARGO_PARAMETERIZATION_CHROMIUM` to an installed Chromium executable when it is not bundled with Playwright.

## Definition authoring (v33)

The resource catalog creates five definition kinds and edits mutable parameter models, forms and parameter bindings.
Published ActionDefinition and WorkflowActionBinding specs remain immutable; clone them to a new name and explicitly rebind.
Editing sends spec only and uses both Kubernetes UID and resourceVersion preconditions.
Validation uses strict server dry-run against installed CRDs under the caller's credentials; it never persists a definition or submits a Workflow.
Execution preview remains responsible for concrete input, reference, implementation and authorization checks.
Concurrent changes keep the unsaved editor contents visible and require rereading before another save.
Status, metadata and immutable execution receipts are not writable through this endpoint.
Named publication/history and cloning are documented above; optional definition status is described below.

`test/parameterization/definitions_ui_e2e.py` creates all five definitions from the actual browser, verifies dry-run and conflict behavior, then fills parameters, saves a revision, previews and executes a native Resource action.
It verifies both the resulting ConfigMap and the frozen form recorded in the submission.

## Definition validation status (v34)

Install `manifests/parameterization/definition-status-controller.yaml` only in the isolated lab after building and importing its `Dockerfile.controller` image.
Its process observes definitions and native templates, with no access to Secrets or Workflow mutation.
Select a saved definition in the resource catalog to inspect `DefinitionValid`, validation errors and observed reference identities.
Changing a referenced model revalidates the form/binding; deleting and recreating it records the new UID.
Refresh the catalog to retrieve current status.
A stale `observedGeneration` is displayed as pending, not successful validation.

Run `test/parameterization/definition_status_e2e.py` with the local URL and kubeconfig, using the Python Playwright environment.
It covers missing references, schema changes, deletion/recreation, native reference changes, namespace RBAC, stable status, browser rendering and real native Resource execution.
It verifies that status writes cannot invalidate an otherwise unchanged execution preview and that definition changes never rewrite completed Workflow nodes or frozen receipts.
The controller and API must be built from this version together; the supported scope and verification limits remain documented separately.

## Secret arrays (v36)

Select `secret-array-demo` in the local parameter page and choose a new Parameter set name.
Enter synthetic values of at least eight characters per row, store each using the encrypted-write button, then save, compare, preview and submit.
Row IDs define stable identities; reorder/add/remove controls manipulate references without exposing saved plaintext.
The native loop uses SecretKeyRef and emits only each row ID and a consumption marker.
Use `test/parameterization/sensitive_arrays_e2e.py` for browser input, real consumption, same-Workflow retry and revoked-reference coverage.

## Runtime sensitive input (v37)

Select `secret-input-demo`, save an empty parameter set, preview and submit.
Open the resulting Workflow and enter a synthetic credential of at least eight characters plus a note in the waiting authorization form.
Store the encrypted version first, then validate and continue the node.
The native suspend output carries a reference, and the downstream container consumes it through `SecretKeyRef` without printing the credential.
Run `test/parameterization/sensitive_inputs_e2e.py` for frozen-source deletion, response-loss recovery, native retry retention and credential revocation checks.

## Runtime option sources (v39)

Install `examples/parameterization/input-options.yaml` and submit an empty parameter set with the `input-options-demo` binding.
At the native suspend, select `test` and `Test East`, then change the environment to `production`.
The previous region becomes unavailable; choose `Production West` and submit to pass `west` to the native container.
`test/parameterization/input_options_e2e.py` covers frozen-source deletion, browser dependencies, revoked choices before commit, missing sources, source-version evidence and native consumption.
The form and provider identity are frozen; allowed choices are rechecked at acceptance using the current caller.
Historical input reads and committed-response recovery do not re-query the source.

### Nested call partial retry (v40)

Run `test/parameterization/nested_retry_e2e.py` for save, native failure, changed revision, Diff, impact preview, native retry, and immutable receipt recovery.
It invokes the same nested DAG twice, retains two approvals and the unaffected loop results, and verifies that only the changed call and its dependent report rerun in the same Workflow UID.
The reopened DAG records current inputs; the receipt retains its previous inputs.
Nested DAG loops, steps, recursion, changed topology, and output-driven loop changes remain explicitly unsupported.

## Artifact, external-secret and topology acceptance

The bounded full paths are `artifact_calls_e2e.py`, `external_secrets_e2e.py` and `topology_retry_e2e.py`.
They use only the isolated local context and fresh synthetic resources.
The artifact case passes actual files through native input/output mappings, fails a downstream operation, and verifies original retry consumes the retained artifact in the same Workflow.
The topology case exercises DAG and Steps expansions with 2, 3, 0 and 1 items while retaining the unrelated approval and successful Pod UID.
The Vault case uses an actual local Vault KV-v2 server, immutable native Secret materialization, old-version retry, revocation, reviewed rotation and backend outage expiration.
The examples `artifact-mappings.yaml`, `topology-retry.yaml` and `external-vault.yaml` install corresponding local-only demos; the Chinese testing guide provides the UI walkthrough.
No real cloud credentials or resources are used.
The Vault development manifest must never be installed in a shared environment.
Production external-secret installations must provide a trusted TLS origin, restricted token file, namespace RBAC, Secret encryption at rest and metadata-only auditing.

## Action replacement review

`action_replacement_e2e.py` exercises author-reviewed effect, target, permission and output semantics through definition creation, preview, rejected submit, native execution and a new-parameter retry in the same Workflow.
It uses only synthetic local string calculations and a digest-pinned Alpine image.
Run it with the same explicit local kubeconfig, context and server URL as the other acceptance scripts.
The resulting `action-contract-*` definitions and Workflow remain visible in the local UI for inspection.

To prepare another action, declare its `spec.executionContract` and retrieve `GET /api/v1/parameterization/{namespace}/action-reviews/{name}`.
Review the returned native implementation, schemas, semantic contract and digest before editing the source leaf's `parameters.workflows.example.io/action-replacement-policy`.
The policy contains `contract`, `implementationDigests`, and `interface` with the exact `inputMappings`, `outputMappings`, `inputArtifactMappings` and `outputArtifactMappings` used by the binding.
An omitted mapping list is an empty list.
This is explicit author review; retrieving review material does not approve or execute anything.
An implementation without author approval, changed target scope or changed mapping is rejected before creating a Workflow.
Original-parameter retry keeps frozen definitions; new-parameter retry rechecks current definitions and retains unrelated successful nodes and human decisions when the native graph is compatible.

## Credential expiry while waiting

`secret_dispatch_e2e.py` exercises save, preview, submission, waiting approval, expiry, rejected task Pod creation, metadata renewal and original-parameter retry.
The Workflow UID and approval are retained, and only one Pod is eventually created with the pinned native SecretKeyRef.
This is a dispatch-time check for opted-in native task Pods, not a guarantee of later kubelet injection time or running-process credential renewal.
Inspect the resulting `secret-dispatch-*` parameter set and Workflow in the local UI.

## Native retry attempt groups

Run `test/parameterization/retry_groups_e2e.py` against the isolated local server.
Its DAG and Steps examples deliberately fail their first native attempt, then retain an unrelated completed retry group while changing another task's parameter.
Inspect the resulting `retry-groups-*` parameter sets and native Workflow history: the original approval and stable task's two Pod UIDs remain, including its first failed attempt.
The changed task and output consumer retry under the same Workflow UID with their new inputs.
A new parameter revision does not replace native retryStrategy or introduce a second workflow scheduler.

## Checked HTTP artifact reuse

Apply `examples/parameterization/artifact-mappings.yaml` for the local synthetic store, then `examples/parameterization/artifact-retry.yaml`.
In the parameter editor choose `artifact-retry-demo`, save `{"mode":"bad"}`, preview, submit and resume the native approval.
The producer succeeds and the consumer deliberately fails.
Save `{"mode":"good"}`, preview changed-parameter retry and apply it to the same Workflow.
Only `consume` should run again; its `digest` output must equal the producer's `checksum`, while the producer Pod UID and approval remain unchanged.

`test/parameterization/artifact_retry_e2e.py` verifies DAG and Steps through save, preview, execution, file tampering, missing storage, rejected retry, restored storage and actual downstream download.
Storage changes are checked again after preview, before applying the retry.
The HTTP store is volatile and only for this isolated local demonstration; deleting its Pod removes its files.

## Native dynamic credential leases

The optional `dynamic-credentials.yaml` example binds a trusted Vault Agent container through `ActionDefinition` and `WorkflowActionBinding`.
The parameter model stores only an immutable Secret reference to a least-privilege bootstrap token.
Each task attempt asks the configured Vault database engine for a fresh PostgreSQL credential; Agent renews that lease while the task runs.
The native output exposes only the opaque lease identifier, so execution history distinguishes attempts without exposing the database username or password.
This is a native action integration, not a built-in Argo lease controller or automatic dynamic-secret support for every execution channel.

The reviewed implementation fixes the backend and allowed role; ordinary parameter values cannot replace either destination or command.
Its policy disables process restart on secret changes, limits task duration below the lease's maximum lifetime, and never falls back to another identity.
The example runs read-only SQL and suppresses credential-bearing database errors.
Completion and native cancellation stop renewal; the remaining lease expires according to its TTL rather than being claimed as immediately revoked.
For workloads requiring immediate revocation or in-process credential replacement, provide a provider-specific native integration with its own explicit cancellation and renewal contract.
Production installations must replace the local dev server and token-file authentication with their reviewed TLS and authentication configuration.

Build `examples/parameterization/dynamic-credentials/Dockerfile` as `argo-parameterization-vault-task:local-v52`, then import that image into the isolated local k3s cluster.
With the same local kubeconfig, context and server environment used by the other acceptance cases, run `python3 test/parameterization/dynamic_leases_e2e.py`.
The bounded path uses a real Vault database engine and PostgreSQL: save, preview, submit, native approval, renewal beyond the initial TTL, explicit revocation, denied acquisition, original-parameter retry, fresh lease, preserved approval, native cancellation and TTL cleanup.
Only synthetic test credentials are used; backend fixtures are removed afterward.

For manual UI testing, run the same script with `--install-demo`.
It installs the isolated backend, definitions and a 24-hour synthetic bootstrap token, but submits no Workflow.
Select `dynamic-credential-demo` in the parameter page and use the following reference:

```json
{"credential":{"secretRef":{"name":"dynamic-credential-demo","key":"token"}}}
```

Save, preview, submit and resume the native approval.
The task performs database reads for approximately 36 seconds and records an opaque lease identifier.
The token expires after 24 hours; reinstalling the fixture is an explicit local administrative action, not an automatic identity fallback.

## Per-call result bindings

See [bound action results](execution-results.md) for the reusable result CR, output mappings, executor-owned observations and frozen retry semantics.
The complete path is `test/parameterization/result_bindings_e2e.py` and includes browser verification.
For step-by-step instructions in Chinese, see [result binding acceptance](execution-results.zh.md).
