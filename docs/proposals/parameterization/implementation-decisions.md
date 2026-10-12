<!-- cspell:words gojsonschema secretbox materializer -->
# Parameterization implementation decisions

Status: experimental, not an upstream API. No deployment to staging or production.

## Baseline and contribution boundary

Initial environment inspection found v3.7.6.
The current isolated lab and release port use v4.0.1.
The separate upstream adaptation is rebased onto main (4192c5bb2bac2b5dea5c1618ea56220fa55a388b, module v4).
Compatibility and upstream design review remain separate acceptance items; neither branch is represented as merge-ready.

Existing delivery repositories are read-only. Only this Argo source checkout is modified. Existing production versions may be copied to staging as a separate operation, with environment-specific credentials and endpoints preserved. Enhanced Argo artifacts are local-only.

## Native ownership

Argo owns Workflow scheduling, task state, suspend/resume, and retry. New definitions resolve to native Workflow specs and native leaf templates. No secondary run/DAG state machine. A submission receipt tracks deterministic Workflow creation, not execution status.

Opt-in experimental API under a separate example API group. The production API group name needs upstream agreement. Argo Server authenticates extension requests through its existing gatekeeper and uses caller-scoped clients. The feature is disabled unless explicitly enabled.

## Schema dialect

Use draft-07, supported by the existing gojsonschema dependency. Advertise the dialect and reject unsupported keywords and remote references; do not claim 2020-12 compatibility. Explicit false/zero/null/empty values survive defaulting.

## History

Create immutable, content-addressed revisions first; advance the ParameterSet pointer with resourceVersion CAS. A revision not selected by a committed set is not presented as a successful save. Keep the source schema with each revision; form, binding, action and native Workflow snapshots are frozen with the execution receipt. Do not use Kubernetes resourceVersion as a history ID.

## Local isolation

Use a dedicated local k3d/k3s cluster, namespace argo-parameterization-lab, namespaced Controller/Server and instance ID parameterization-lab. Native CRDs are installed only in this new cluster. The initial namespace on Rancher Desktop was retired after export; pre-existing local Argo workloads and CRDs retain their original specs.

## Retry

Native v3.7.6 retry retains Workflow UID and supports workflow parameter overrides. It also resets other failed execution nodes. It is not an isolated arbitrary-node retry guarantee. A parameter revision change requires an impact preview; preserved successful inputs must not change. Unsupported active Workflow, topology or completed-side-effect changes fail explicitly. Admission and authenticated APIs must prevent a forged snapshot or not reviewed binding.

## Sensitive inputs

Use Secret references, never plaintext Workflow arguments. Encrypted input requires a dedicated write endpoint, authenticated encryption, separate key storage and bound namespace/context. Do not describe Kubernetes base64 as encryption. Unsupported execution channels must reject sensitive mappings.

## Acceptance

Each implemented subset needs UT and real local-k3s E2E. Unsupported capabilities must be reported, not silently accepted. Cloud providers in demos are isolated test services; no real delivery endpoints, credentials, account IDs or business names are embedded.

## Resource discovery and native entry

The opt-in catalog lists the eleven experimental resource kinds using caller-scoped
Kubernetes clients and pagination. It is not an arbitrary Kubernetes proxy and
does not list Secret objects. Native WorkflowTemplate details and submit panels
discover compatible namespaced bindings and link to an explicit selection in the
structured editor. Opening a binding performs reads only. Templates without a
binding, and servers without the extension, retain the native submit entry.
Top-level ClusterWorkflowTemplate binding uses explicit clusterScope and caller RBAC.
Opt-in Workflow admission shares validation with the API. Initial native submission requires a frozen submission whose spec matches the resolved definitions and parameter revision. The optional webhook checks actual caller RBAC before dependency reads and does not impersonate users. It is independently installed and does not intercept unrelated native Workflows; the discovery UI is not an admission control.

## Encrypted input and intermediate forms

The opt-in encrypted Kubernetes backend requires API-server encryption at rest and a metadata-only audit policy for Secret writes. Local k3s uses the built-in secretbox provider (XSalsa20/Poly1305), with native key rotation verified. The backend adapter uses caller credentials, immutable Secrets and random request identities; it returns references, never plaintext or ordinary plaintext hashes. Other stores may implement SecretWriter; the optional Vault KV-v2 materializer is described below. HTTPS is required except an explicit loopback-only development option. Latest-at-submission reference indexes are described below; array references are described in v36 below.

Intermediate forms are frozen with the execution receipt and bound to native suspend templates. Validated non-secret values become supplied native outputs via SetWorkflow. Immutable input receipts plus Workflow UID/resourceVersion preconditions recover ambiguous responses without resuming another node. The controller remains the only scheduler. Sensitive plaintext outputs are rejected; v37 adds reference-only sensitive input and v39 adds frozen intermediate option sources below.

## Checked new-parameter native retry

A completed frozen DAG, including statically named nested DAG calls, can opt in to partial retry.
Changed workflow parameters are mapped to task arguments and their dependency closure.
Changing loop identity, conditions, templates, native Secret environment bindings, exit hooks or artifact reuse is rejected until an explicit proof is implemented.
Unchanged successful tasks must declare `parameters.workflows.example.io/result-reuse: same-inputs`; unchanged native suspend decisions are preserved.
Affected non-suspend implementations must declare `parameters.workflows.example.io/retry-safe: idempotent`.
This is an author contract, not a claim that arbitrary external side effects become idempotent.
The server previews native retry and checks the input/output/result fingerprint of each retained invocation before allowing it.
Graph adjacency can be recomputed by native Argo; execution inputs, outputs, phase and timestamps cannot change.
A ParameterRetry freezes the previous Workflow and selected new revision before native retry, and the Workflow UID remains unchanged.
Concurrent changes invalidate the preview; ambiguous responses recover through the immutable receipt and native retry annotation.
Unsupported cases never fall back to creating a new Workflow.

## Action execution contracts

The resolver freezes input/output schemas as inert template annotations alongside the native implementation.
The native controller validates resolved inputs before dispatch and validates synchronized Pod or Agent results before success and downstream consumption.
No extra workflow or validation scheduler is introduced.
String properties use native string values; explicitly typed numbers, booleans, objects, arrays and null use JSON encoding.
HTTP and script implementations can declare the native `result` field in their output contract.
Schemas with ambiguous top-level encoding, memoization, daemon tasks, plugins or composite action implementations are rejected in this subset.
Named artifact adapters use the native interface described below; artifact contents are not JSON parameters.
Schema errors identify paths without echoing input/output values.
Older unannotated native templates retain their behavior; deployment requires the enhanced Server and Controller together.
Input/output contracts do not by themselves guarantee external idempotency or recovery; ResourceOperation uses the optional single-operation recovery library and reference adapter described below; provider-specific production adapters remain separate.

## Single-operation recovery reference

The optional library persists a CAS-protected dispatch claim before an external
request. A timeout or lost response triggers observation of the same logical key;
a missing receipt after dispatch blocks further mutation as OutcomeUnknown.
Outputs are schema-validated, and original retry preserves the operation key.
The reference operator checks the materialized native Resource Pod and frozen
submission action version. It never writes Workflow status or dispatches another
task. It supports a synthetic HTTP provider only, with explicit namespace and
connection configuration. This is not a production connector or an arbitrary URL
execution API. Stop/terminate/deletion can request cancellation or detach; suspend
does not cancel. Operation records remain separate from Workflow garbage collection.
See examples/parameterization/resource-operation/README.md for limits and bounded
real-k3s tests. Provider adapters and general nested/loop
identity guarantees remain separate acceptance items.

## Native submission enforcement

A separately installed, namespace-scoped HTTPS validating webhook shares the API validator.
It accepts only frozen, typed inputs on opted-in creates, and requires a checked ParameterRetry for changes to execution inputs.
The caller must be able to read every dependency; SubjectAccessReview carries the actual admission username, groups and extras.
The webhook has read-only definition credentials and permission to create SubjectAccessReview, but no impersonation or Workflow-write permissions.
Draft/definition authorship remains governed by Kubernetes RBAC; this is not a replacement for permissions on creating models or immutable snapshots.

Native state reconciliation, suspend/resume and stop/terminate do not re-resolve mutable definitions.
Checked partial retry verifies the prior Workflow identity/version, recomputes the supported impact plan and preserves successful invocation fingerprints.
Direct parameter edits and removal of the original contract annotations are rejected.
The installation excludes unannotated native workflows and uses failurePolicy Fail for adopted objects.
The webhook needs the same configured option-provider adapters as the API; the reference install currently supports Kubernetes sources only.

## Matching-release compatibility (v4.0.1)

The enhancement has also been ported to official v4.0.1 for the isolated lab;
the remote installations continue to use the official unmodified release.
The community branch is rebased on upstream main separately. These are explicit
compatibility targets, not a claim that the entire design is complete.

The native controller resolves spec expressions into a separate execution view.
Persisted input expressions remain intact for admission and subsequent retries,
while the native node-state map remains shared with the execution view. Native
shutdown tests change the submitted spec, as the public API does, rather than
mutating a derived execution view. Admission initializes the logger context needed
by v4 native retry and reads parameter models needed by intermediate forms.

## Recovering unsuccessful Workflow status writes

A terminal result in controller memory is not a committed terminal Workflow.
If the Kubernetes update fails, the controller retains its parallelism slot and requeues the same Workflow for a fresh read.
Pod results and operation receipts remain available until the terminal write succeeds.
This recovers transport and admission outages without resubmitting the Workflow or invoking a native task retry.
The v4.0.1 port adopts the running-workflow and in-memory slot protection present in upstream main, with explicit requeue on unsuccessful persistence.
The local acceptance case rejects the synthetic Workflow's completion using a temporary namespace-and-name-scoped webhook, restores availability, and verifies the same Workflow, Pod and output ConfigMap UIDs converge without a controller restart.
The admission server sets an explicit 90-second idle keep-alive timeout rather than inheriting its 10-second request-read limit.

## Declarative presentation semantics

WorkflowForm is consumed directly by both the initial parameter editor and frozen native input forms.
Layout nodes describe nesting and field groups; they never create workflow dependencies.
Field paths are JSON Pointers, with numeric-array wildcards resolved against the current row.
Hiding or disabling a field preserves its saved value and does not bypass schema validation or Kubernetes authorization.
Malformed graphs, missing field references and widget/schema mismatches are rejected in preview and reported in the UI.
Option source aliases resolve to the same caller-scoped providers and are revalidated at preview.
Tabs preserve editor state and invalid YAML prevents persistence until corrected.
Intermediate option providers use the frozen-node contract described in v39 below.
Sensitive plaintext outputs are rejected; runtime forms support immutable references through explicit `secret-reference-json` mappings.

## Definition spec authoring

The catalog authoring endpoint is limited to the five definition kinds and uses the current caller's Kubernetes transport.
It never accepts arbitrary kinds, metadata, status, parameter sets or execution receipts.
Validation is strict Kubernetes server dry-run; updates require both the original UID and resourceVersion, preserving metadata and status through normal subresource semantics.
Published action definitions and bindings remain immutable and are cloned into a new name.
The endpoint does not claim to publish a resolved execution contract: preview and submit still freeze and validate references and actual values.
Named immutable publication and definition status reconciliation are described below.

## Definition status ownership (v34)

The optional namespace-scoped definition controller observes only the five editable definition kinds and native template references.
It uses list/watch caches and reverse-reference indexes, then writes only their status subresources with resourceVersion preconditions.
Its separate service account cannot read Secrets or mutate definition specs, Workflows, execution receipts or external resources.
It does not schedule, resume, retry, approve or change any Workflow status.
The local manifest separately grants read-only access to native ClusterWorkflowTemplates; that scope is disabled by default in the binary.

`DefinitionValid` reports supported static schema, form, mapping and reference checks for `observedGeneration`.
A form without a declared model remains `Unknown/BindingContextRequired` until evaluated in a binding context.
Missing/replaced definitions, changed schemas and missing native call targets invalidate dependents and recover after correction.
Dynamic options, credentials, concrete values and native execution validation still require preview/submit with caller-scoped clients.
The condition is advisory and is never accepted as an authorization decision or proof of prior execution.

Resolved references retain UID, observed resourceVersion and spec digest.
Status-only resourceVersion changes do not generate reconciliation feedback or change content identity.
New execution content identities for extension definitions use UID plus spec digest; editing still uses UID/resourceVersion CAS.
A change to the UID or definition contents invalidates a preview, while changing validation status alone does not.
Historical receipts retain the original definition version strings and snapshots and are never migrated in place.

## Named definition publication (v35)

An explicit publication creates a new object of the same definition kind with its reviewed spec.
This preserves existing typed CR contracts and name references without another run resource or version controller.
The source UID and spec are preconditions; status-only writes do not invalidate publication.
The publication name is the idempotent receipt, including after a lost response or source deletion.
Publication never changes the source, redirects a binding or starts a Workflow.

Reserved `parameters.workflows.example.io/definition-*` metadata records source identity, spec digest, family UID, publication time, publisher and note.
The installed immutable-snapshot admission policy protects the published spec and its provenance on direct Kubernetes updates too.
These metadata are descriptive author provenance, not an authorization grant; Kubernetes authors and administrators still control resource creation and deletion.
Existing draft definitions remain mutable, while action definitions retain their previous immutability.
The API rechecks static schema/reference contracts using caller-scoped reads before creating a publication.
An unbound reusable form may still require a concrete model at binding validation.

Versions are listed by a Kubernetes family-label selector with 25-item continuation rather than by scanning all definitions.
Bodies are opened individually; semantic spec comparison uses bounded, content-bound pages and rechecks read permissions on every call.
Restoring is an explicit clone into a new name, followed by an explicit binding selection.
Neither old publications nor saved parameter/execution snapshots are overwritten.
This versions one definition, not an atomic release of every transitive dependency.
Authors can reference published names in other definitions; full reference closure and native templates are still frozen at execution preview/submission.
Independent retention/deletion policy and a multi-definition release bundle remain outside this subset.
Unpublished edits remain drafts; named publications have independent, queryable history.

The editor handles one metadata/status-only save conflict by reading the current object and verifying identical UID, generation and original spec before retrying once with its current resourceVersion.
A changed generation, changed spec, replacement UID, failed read or second conflict is surfaced without losing the user's unsaved text.
The API itself retains strict Kubernetes CAS semantics; concurrent edits are never silently merged.

The reference operation adapter accepts both legacy UID/resourceVersion receipts and UID/spec-digest receipts.
For content identities, ActionDefinition immutability and the already-verified native Pod manifest bind the operation; an observed status resourceVersion is not an execution version.
The frozen typed Argo template may serialize omitted fields explicitly, so its wire encoding is not substituted for the captured original source digest.
The adapter continues to reject replaced UIDs, mismatched explicit digests, absent frozen actions and malformed content identities.

## Sensitive arrays and native execution (v36)

Sensitive paths may use numeric array wildcards, for example `/accounts/*/credential`.
Every wildcard array boundary requires a ParameterTemplate identity rule; malformed collections, duplicate identities and plaintext secret fields are rejected before saving.
The same matcher validates API writes, declared secret widgets, native mappings and reference-only Diff, including nested arrays and escaped pointers.
Derived mappings overlapping sensitive paths are rejected rather than dropping their sensitivity.

The dedicated encrypted-write endpoint accepts concrete row paths and returns immutable Secret references.
The browser disables form mutation while a write is pending, clears unsaved plaintext when a row moves or changes identity, and discards late responses for replaced/unmounted fields.
Saved references move with the row's data.
Ordinary JSON mappings still reject sensitive paths and their ancestors/descendants.
An explicit `secret-reference-json` encoding permits only complete reference objects or an enclosing document; it never retrieves or serializes plaintext into arguments.
Preview and submission authorize every consumed Secret, require immutability and the declared logical schema, and capture each UID/resourceVersion.
Authors may feed these reference documents into native `withParam` and native `secretKeyRef`; Argo still owns loop expansion and execution.
Nested item reference properties use native expressions such as `{{=jsonpath(item.credential, '$.secretRef.name')}}`; simple item tags expose only the first property level in the pinned native version.
Original-parameter retries retain the same Workflow and recheck the pinned Secret identities without resolving newer definitions.
A revoked or replaced Secret fails explicitly instead of being silently substituted.

`test/parameterization/sensitive_arrays_e2e.py` writes three synthetic credentials through the real browser/backend, reorders and deletes rows, compares versions, runs native Secret-consuming loop containers, recovers a deliberately failed downstream Resource task in the same Workflow, and checks reference-only API/history/receipt/log results.
Sensitive intermediate suspend inputs are covered in v37 below; latest-at-submit credential indices are covered in v44, while external secret-store implementations remain separate work.

## Frozen runtime credentials (v37)

Native suspend forms may declare sensitive fields with `secret-reference-json` output mappings.
The dedicated encrypted-write endpoint accepts either a live parameter template or a Workflow UID, waiting node ID and frozen contract digest, never both.
Runtime writes resolve the immutable submission contract even after its source model and form are deleted.
The supplied native output and `ParameterNodeInput` contain only references, provenance and immutable Secret version identities.
The caller must still be authorized to read the Workflow and frozen receipt, and create/read the Secret.
An already-applied input request can be recovered without resuming the node again, even if its credential was subsequently revoked.
An uncommitted input checks access, logical schema and immutable Secret identity before native `SetWorkflow`.
Native API and admission retry resets recheck frozen submission and input Secret identities, without resolving mutable definitions.
Ordinary controller status writes, suspend/resume and completion do not repeat those checks.
This validates the submission/retry boundary; Kubernetes `SecretKeyRef` does not itself pin a Secret UID, so the namespace's delete/recreate permissions remain part of the credential trust boundary.

## Exact parameter dependency references (v38)

Partial retry reads native template tags and Expr member access instead of searching for parameter-like text.
Static dot and bracket references preserve exact parameter names, including names with a common prefix.
Plain documentation and expression string literals do not create dependencies.
Whole-object or dynamic-key access conservatively depends on every parameter in that scope; dynamic `$env` access is rejected.
Object keys are inspected because native substitution also processes serialized keys.
Observed invocations must match their native boundary, frozen task/template identity and declared loop form.
Preview still checks native retry's resulting retained-node fingerprints; this analysis does not schedule tasks or broaden the supported topology.
The full-path test changes one parameter, retains a similarly named parameter's three loop results and a suspend approval, then performs a new-parameter retry followed by an original-parameter retry on the same Workflow UID.

## Frozen intermediate option providers (v39)

Runtime form lookups identify the Workflow UID, native waiting node and immutable input-contract digest, mutually exclusive with a live binding.
The provider declaration comes only from the frozen input form; deleting or changing mutable model and form sources does not redirect the lookup.
Providers run with the caller's current credentials, bounded query time and existing source limits.
Dependencies filter choices through the same semantics used by initial forms; the UI discards responses after values or input identity change.
Before completing the suspend node, the server re-reads allowed choices and rejects unavailable or disabled selections.
The immutable ParameterNodeInput records source and dependency versions with validated values.
A receipt whose native write did not happen must revalidate its original source evidence; changes require a fresh reviewed input request.
Recovering an already-committed response or reading submitted inputs does not re-query options or repeat native execution.
Permission errors remain explicit even when a source allows manual input.
Provider freshness is checked at input acceptance, not an external reservation or a guarantee that an external resource remains available indefinitely.

### Nested partial retry (v40)

Impact analysis follows named DAG call paths and propagates changed input names through arguments.
Two invocations of the same template have distinct analysis contexts.
Dependency closure remains conservative; a changed upstream call also invalidates its dependent tasks.
Only native Argo computes and persists the retry reset, with a checked retained-result fingerprint set.
Completed unrelated boundaries, leaf results, and approvals must survive that native reset unchanged.
Native retry preserves a completed predecessor TaskGroup when none of its children is reset or deleted; retrying an actual loop child still resets its group.
A reopened native DAG records the freshly processed arguments it actually uses, while completed child input evidence stays unchanged.
Changed conditions or output-dependent loop identities are rejected rather than treated as unchanged topology.
This increment does not support nested DAG loops, steps, recursive calls, changed template structure, artifact reuse, or retry groups.

## Referenced native action implementations

ActionDefinition supports exactly one of an embedded `template` or an `implementationRef` using native `name`, `template` and optional `clusterScope` fields.
A namespaced reference resolves in the binding namespace; no cross-namespace impersonation or arbitrary provider lookup is introduced.
Preview and submit read the library using the caller's Kubernetes client, freeze the native leaf and its pod metadata, and record the library UID, resource version and content digest.
As with native templateRef, the invoking Workflow supplies Workflow-level arguments, defaults, service account and execution policy; the library does not replace them.
Both the API and definition controller validate the resolved leaf against the action input/output schemas.
The CRD requires exactly one implementation, and unavailable, forbidden, dynamic, absent or unsupported implementations fail before execution.
Concurrent changes to a library invalidate the preview; mixed references cannot freeze two versions of the same source in one execution.
The immutable action definition retains its reference, while the submission separately retains the actual native implementation and source identity.
Deleting or updating a source after submission does not alter the running Workflow or original-parameter retry.
This increment does not extend template-level action bindings into per-invocation mappings, artifact interfaces or plugins.

## Named action call sites and native interface adapters

New `WorkflowActionBinding.calls[].callPath` values start with the native entrypoint template, followed by named DAG tasks or steps.
The parent template opts in with `parameters.workflows.example.io/bindable-calls: '["emit"]'`.
A terminal loop site applies to every iteration; descending through an intermediate loop or recursive ancestor is rejected.
The compiler clones only the selected invocation ancestors into the immutable Workflow and adds a one-child native DAG interface adapter.
Shared source templates, other invocations, dependencies, conditions, loop expressions and call arguments remain unchanged.
`inputMappings` maps baseline parameter names to action input names; `outputMappings` maps action outputs back to baseline names.
Mappings are complete one-to-one native wire mappings, not an expression language or another scheduler.
Non-string action fields use the JSON encoding declared in the action schema.
Missing, duplicate, overlapping and unapproved bindings are rejected before submission.
A baseline with implicit result/exitCode, daemon, memoization, retry-group or composite execution semantics cannot use the interface adapter.
Artifact inputs and outputs use separate native artifact mappings described below.
Script/HTTP action results may map to explicit baseline output parameters.
Template-level Secret environment bindings and runtime input forms cannot be silently transferred to a replacement action.
Legacy whole-template bindings remain readable and executable, but cannot mix with call-path bindings or new interface mappings in one binding.
Author-reviewed effect/capability compatibility is described below; production provider adapters remain separate acceptance items.

## Latest-at-submission Secret policy

`secretRef: {name, key}` identifies a fixed immutable Secret version.
`latestSecretRef: {configMap, key}` selects a same-namespace ConfigMap entry containing only `{name, key}` for an immutable Secret version.
The parameter model must explicitly permit this reference shape.
The index and the target Secret are both read with caller credentials; the index grants no access to its target.
Preview creates no resources and returns no plaintext or plaintext digest.
Submit re-resolves the index and rejects a changed execution digest, including rotation after preview.
The execution records index UID/resourceVersion and the actual immutable Secret UID/reference; Kubernetes resourceVersion is never treated as a historical secret store.
Original-parameter retry keeps the pinned version and still checks current authorization and revocation.
A reviewed parameter retry using reference-only JSON arguments can resolve the index again, even when the saved ParameterRevision did not change.
The native Workflow UID and unrelated results remain unchanged; the retry receipt records the previous and next public version identities.
Already committed submission, input and retry receipts recover their pinned version without reading a newer index.
Runtime input resolves the version when the user explicitly submits that native input, then freezes the reference in its receipt.
Automatic native retry pins materialized Secret versions. The separate native Vault Agent action below obtains a renewable database lease per task attempt; the server itself does not issue leases.
A template-level Secret environment binding cannot rotate through the current parameter-only retry API because that would alter the frozen native template.
External secret backends retain the SecretWriter adapter boundary; the optional registered Vault materializer is described below.

## Extended native partial retry

The impact checker accepts frozen DAG and Steps entrypoints, nested DAG/Steps calls and fixed-identity composite loops.
Steps ordering remains entirely native; only actual output consumers propagate input changes across sequential groups.
Observed node identities include the native Steps group and loop iteration, rather than matching only template or display names.
Completed unaffected groups, approvals, Pod UIDs and input/output fingerprints must survive a dry-run of native retry.
Reopened Steps boundaries record processed inputs for the new attempt, as DAG boundaries already do.
Changes to the frozen template graph, conditions, artifacts, memoization or composite retry groups remain explicitly unsupported.
Native retryStrategy groups around container, script and resource leaves are checked as described below.
Author-approved whole-loop replacement is described below.
No unsupported case falls back to resubmitting a whole Workflow.

## Community acceptance boundary

The feature remains opt-in and uses an experimental API group. Public design review is tracked in [issue #17221](https://github.com/argoproj/argo-workflows/issues/17221); maintainer approval is not implied.
The contribution checklist requires an upstream issue/design discussion, the final API-group decision, a feature entry tied to a real issue, DCO and AI disclosure.
Do not invent an issue number or describe local tests as upstream maintainer approval.
Provider-specific integrations and the unsupported combinations above remain explicit extension boundaries, not validated capabilities.
Both the current upstream branch and the matching-release port require their own affected-package and native regression checks.
`make pre-commit -B` is a separate required check; failures and unavailable tooling are recorded, not hidden by the local E2E results.

## Feature acceptance ledger (v47)

This ledger distinguishes tested implementations from unsupported extensions and external community decisions.
The local release uses official v4.0.1 as its compatibility base; the separate community branch targets upstream main.
Neither enhanced branch is deployed to pre-production or production.
The capabilities endpoint publishes named action, credential-version and native retry scopes along with explicit unsupported combinations.
A successful capability lookup does not authorize an operation or replace concrete preview checks.

| Feature area | Implemented and tested | Remaining boundary |
| --- | --- | --- |
| Typed resources | Eleven experimental typed CRDs, separate spec/status, immutable revisions and execution receipts, definition conditions | API group and public lifecycle require upstream design agreement |
| Types and defaults | Draft-07 schema, explicit supported keywords, false/zero/null preservation, schema-local type references, structured resolver | Cross-model type-library publication and draft-2020-12 are not claimed |
| Form and binding | Same model for draft and suspend input; layout, arrays, conditions, bounded option providers, YAML/JSON, native arguments | New external providers and controls require adapters |
| History and Diff | Immutable save versions, CAS, restore-to-new revision, paged structured/text and execution-layer differences | Multi-definition atomic publication is not implemented |
| Execution closure | Caller-scoped definitions, static templateRef closure including ClusterWorkflowTemplate, digest revalidation, idempotent native submission | Dynamic template references and arbitrary global Plugin replay are rejected |
| Action calls | Legacy whole-template binding and opt-in named DAG/Steps call paths; parameter and native artifact interface adapters; input/output checks; frozen implementations; author-reviewed execution contracts | Automatic proof of arbitrary program effects and per-iteration call selection are unsupported |
| Operation recovery | Synthetic external operation adapter, persistent identity, timeout/response-loss recovery, cancellation semantics and result validation | Real provider integrations are adapter work, not demonstrated cloud support |
| Secrets | Immutable references, encrypted write seam and local implementation, dynamic arrays, runtime inputs, latest index resolution, Vault KV-v2 materialization, rotation/revocation and redaction checks | Native Vault Agent action covers renewable per-attempt database leases; arbitrary direct execution-channel injection remains unsupported |
| Partial retry | Native original retry and reviewed new-parameter retry in the same Workflow; DAG/Steps, static nested calls, stable composite loops and author-approved whole-loop shape replacement; approvals/results retained | Arbitrary template topology edits, per-item loop result reuse, unchecked artifact reuse, non-HTTP artifact stores, memoization and composite retry-group expansions remain blocked |
| UI and API | Integrated Argo navigation, binding discovery, saved/actual inputs, native nodes, operation receipts and current OpenAPI proposal | Public generated gRPC/CLI API integration and the final upstream UX contract remain |
| Examples | Data-processing, training, external-operation and high-dynamic behavior fixtures; new action, credential-rotation and Steps demos installed locally | Examples do not prove compatibility with a specific production delivery system |
| Community | Separate mainline branch, signed feature commit, native generator and regression checks, compatibility port | Maintainer discussion, API review and complete upstream CI are still required; issue #17221 and a feature note now track the proposal |

The new full-path cases use actual local Argo Server, admission, controllers, CRs and Pods.
Steps retry retained two approvals and five successful Pod UIDs; nested-loop retry retained three approvals and seven successful Pod UIDs.
Named-action cases verified real mapped outputs while leaving the unbound sibling implementation unchanged.
Credential rotation verified that original retry uses the old version while reviewed parameter retry uses the new immutable version in the same Workflow.
Browser input tests cover latest-reference submission and recovery from a committed receipt after the index disappears.
Prior completion-write, external-operation and original-retry regression evidence remains part of the local acceptance history.
These results validate the supported implementation scope, not optional extensions or the entire upstream CI suite.

## Community review package

Proposed title: `feat: add opt-in structured workflow parameters and action bindings`.
The motivation is reusable typed input, declarative forms, versioned execution inputs and explicit native action contracts without a second workflow scheduler.
The implementation keeps original workflows opt-in, fixes definition and credential identities at reviewed submission, and delegates scheduling and retries to Argo.
The v4.0.1 tag checkout is retained for local compatibility testing. Public PR bases are main and release-4.0; a Git tag cannot be a PR base. The release branch has its own native regression and local execution checks.

Verification evidence includes resolver/server UT, UI tests, native retry/controller regressions and isolated k3s full paths with real Pods and persistent receipts.
Native generation, full mainline Go/UI lint and strict documentation targets passed locally. The initial pre-commit command hit host-tool compatibility errors; its failing targets were rerun after correcting the tool environment. Full upstream CI remains separate.
Review should explicitly cover the experimental API group, RBAC/admission boundary, schema dialect, snapshot lifetime, native retry changes and adapter capability limits before treating this as a mergeable feature.
The contribution references public design issue #17221 and a corresponding feature note. The implementation preceded upstream discussion, which is explicitly disclosed in that issue.
The feature commit is signed off and remains a single local feature commit per compatibility branch.
Code, tests and documentation were developed with OpenAI Codex assistance; a public PR must include that AI disclosure and a human review statement based on actual review.

## Local v46 verification record

The compatibility deployment is `argo-parameterization:local-v46.1-v4.0.1` in the isolated local k3s namespace.
The capability endpoint reports named action paths, latest-at-submission secret indices and DAG/Steps nested retry scopes.
The deployed execution implementation passed seven distinct full-path suites: action calls, referenced actions, credential rotation, Steps/composite-loop retry, external-operation recovery, browser latest-secret input and browser frozen sensitive inputs.
The external-operation suite includes lost responses, controller restart, original retry, cancellation, unknown outcomes, invalid outputs and forged invocation rejection.
The compatibility and mainline branches passed the affected Go packages and native retry/controller regressions; the mainline cleanup was followed by another affected-package run.
The UI parameterization suite passed 58 tests in 14 suites; TypeScript and the modified UI ESLint scopes passed.
Mainline parameter packages passed the repository Go lint configuration, and the remaining modified native integration paths passed lint against the upstream base.
Native mainline generation successfully produced typed clients, informers, listers and deep-copy methods for the experimental resources; CRDs remain a separate opt-in install.
These scoped checks do not replace the still-incomplete full upstream `make pre-commit -B` and CI run.
The earlier whole-repository lint attempt exceeded its local resource/time budget; the historical full Go suite also has a recorded local timeout and a Darwin-specific artifact-plugin failure.
The final 300-second `make pre-commit -B` attempt completed native and Java generation but reached its bound during `go mod tidy`, before full lint and documentation acceptance.
At this historical checkpoint no public issue, feature note, maintainer approval or PR existed; the community review package above records the current contribution status.

## Native artifact mapping

Named action calls expose separate `inputArtifactMappings` and `outputArtifactMappings`.
Mappings must be complete one-to-one native artifact interfaces and cannot turn an optional value into a required one.
The action owns local file paths, storage and archive configuration; the adapter owns public names and global exports.
The adapter only passes `inputs.artifacts` and `tasks.invoke.outputs.artifacts` references, never artifact contents or synthesized locations.
Legacy whole-template bindings retain their native interface semantics and cannot specify call-path mappings.
The local test uploads a real generated file through the native HTTP artifact driver, fails a downstream action, then uses original retry and the retained artifact in the same Workflow.
New-parameter artifact reuse uses the explicitly checked HTTP subset described below; other storage locations remain rejected.

## Registered external Secret backend

The optional `parameterization-secret-controller` implements a Vault KV-v2 adapter, configured by an administrator-owned file and a mounted token file.
Vault origin, mount, entry, destination namespace and field are outside user parameters; a workflow cannot choose an arbitrary backend URL.
HTTPS and certificate verification are required except for an explicit development flag used only by the synthetic local demo.
Requests do not follow redirects, have bounded duration/size and redact provider error bodies.
The adapter reads exact version numbers and materializes immutable native Secrets, plus the existing reference-only `latestSecretRef` ConfigMap index.
This keeps native Secret environment/file injection and Kubernetes RBAC, without adding a workflow scheduler or putting values into ordinary CRs.
Kubernetes encryption at rest and metadata-only auditing remain installation prerequisites; materialization is an explicit copy of a secret into Kubernetes, not direct Vault injection.
The Vault API contract is [KV-v2 version and metadata reads](https://developer.hashicorp.com/vault/api-docs/secret/kv/kv-v2).
Rotation advances the index, while original retry retains its previously pinned immutable Secret identity.
Destroyed, deleted or expired versions are removed after authoritative metadata observation; an unavailable backend cannot renew verification indefinitely.
The adapter renews a 60-second metadata verification window every 15 seconds and removes expired injection targets; parameter preview, submission, runtime input and retry reject expired verification metadata.
This bounds extension submission and retry validation while the backend is unavailable.
Removal of native injection targets requires the materializer to keep running; Kubernetes itself does not interpret the verification annotation.
For an opted-in parameterized Workflow, native task Pod dispatch rechecks external verification metadata on concrete environment, volume and image-pull Secret references.
Expired metadata prevents creation of a new task Pod, even if the Workflow was submitted before expiry and waited at an approval.
Ordinary native Workflows are unchanged; existing Pods and successful nodes are not invalidated by a later expiry.
An informer miss can recover an existing Pod only when its controller owner matches the current Workflow UID.
This check runs before API creation, not at later kubelet injection; a created Pod may wait for scheduling past the validity window.
Strict injection-time expiry, agent HTTP/Plugin dispatch and running-process renewal still require separate integration and are not claimed here.
The materializer cannot erase a credential already copied into a running process environment.
The native Vault Agent integration below covers per-attempt database credentials and renewal. Direct HTTP/Plugin credential injection remains outside this integration.

## Parameter-driven topology retry

An author may list named calls in the parent template annotation `parameters.workflows.example.io/retry-loop-calls`.
For those calls, a changed parameter driving `withParam`, `withItems` or `withSequence` can replace the entire native expansion in the same Workflow.
The impact plan selects the old group and its invocations, plus dependent consumers, then verifies native retry preserves unrelated results and approvals.
No changed array item reuses an old positional result, including reorder, insertion, removal and empty-to-nonempty transitions.
Every affected executable leaf still needs the explicit idempotent retry contract; hooks, memoization, artifacts and composite retry groups keep their existing rejection boundaries.
Removing an item does not delete any external resource created by its old invocation; durable side effects need their own reconciliation contract.
A change to the frozen Workflow template graph itself remains unsupported rather than silently resubmitting a new Workflow.

## Local v47 verification record

Both compatibility and mainline parameter packages passed UT for native artifact mappings, Vault KV-v2 materialization and author-approved loop replacement.
Five local full-path suites passed: artifact calls, external secrets, topology retry, named action calls and latest Secret versions.
The artifact case uploads and downloads real bytes through native artifact drivers, deletes the source action after submission, and preserves the producer result through original retry.
The Vault case verifies actual version rotation, pinned original retry, revocation rejection, reviewed parameter retry, redaction and provider-outage expiration.
DAG and Steps topology cases change the loop size from two to three to zero to one, preserving the Workflow UID, unrelated successful Pod and original approval.
These suites ran against only the isolated local k3s cluster with synthetic resources.
The local server image is `argo-parameterization:local-v47.1-v4.0.1`; the capability report lists both the added features and their unsupported boundaries.
Three persistent examples are installed for manual testing: `artifact-mapping-demo`, `external-secret-demo` and `topology-retry-demo`.

The public contribution plan has two targets: upstream main and the upstream 4.0 release branch.
Official `v4.0.1` is a tag, so it cannot itself be a pull-request base; the matching series branch is `release-4.0`.
The compatibility checkout remains pinned to v4.0.1 for repeatable testing, independently of the eventual release-branch port.
No public PR is claimed until the development checklist and contribution checks are complete.

## Author-reviewed action replacement contracts

An optional `ActionDefinition.spec.executionContract` declares `effect`, static `targetScope`, `permissionScope`, `outputMeaning` identities and named capabilities with applicability conditions.
Capabilities are `Idempotency`, `Observe`, `Cancel`, `Recover` and `ResultReuse`; they do not automatically enable native retry or grant target permissions.
A source leaf opting into this contract stores `parameters.workflows.example.io/action-replacement-policy` as JSON with a `contract`, an `interface` containing the four parameter/artifact mapping lists, and an `implementationDigests` list.
The source workflow author controls this policy; putting a policy on the replacement action cannot approve itself.
Named calls still require the existing parent `bindable-calls` declaration.

`GET action-reviews/{name}` returns the resolved action, input/output schemas, declared semantics, source versions and its SHA-256 digest for review using the caller's Kubernetes permissions.
The digest covers the effective native template and contract, ignoring only the compiler-assigned template name and the already-resolved reference address.
Container, script, init-container and sidecar images in a reviewed action must use immutable SHA-256 references.
Changing implementation content, schemas, service accounts, scopes or capability conditions invalidates the previous review.
A native Kubernetes write verb or unsafe HTTP method cannot claim `ReadOnly`, and neither request channel can claim `Pure`.
The candidate must retain the baseline effect, target/permission scope and output meaning, and every required capability condition.
Only an explicitly approved implementation digest and the exact reviewed interface mappings may replace that baseline.
Definition validation and concrete preview enforce the same checks; submission re-resolves the closure and checks its execution digest.

This is an explicit author-reviewed compatibility contract, not automatic proof of arbitrary container code or remote service behavior.
The target system, network policy, service account and native admission remain responsible for enforcing access.
The workflow author must review program behavior, actual target interpretation, output meaning and applicability conditions before approving a digest.
The digest cannot prove the behavior of a remote service or mutable external data that a reviewed program reads.
Both contract and source policy are required if either is present; old definitions with neither retain legacy behavior and are not reported as semantically verified.
Previously frozen submissions and original-parameter retries retain their original reviewed snapshot.
New-parameter retry resolves and validates the current definitions before computing the affected native nodes.
The full local path `action_replacement_e2e.py` covers missing approval, target mismatch, stale referenced implementation, rejected submission without a Workflow, actual output, and same-Workflow retry preserving a human decision and an unrelated successful Pod.

## External-secret task dispatch acceptance

`secret_dispatch_e2e.py` saves and submits a valid external Secret reference, expires its metadata while native approval waits, then verifies that resuming creates no task Pod.
After metadata renewal on the same immutable Secret UID, original-parameter retry succeeds in the same Workflow and retains the completed approval.
The test checks native SecretKeyRef consumption, redacted errors/history/receipts and stable completed status after later expiry.
It uses synthetic values and controlled expiry in the isolated local namespace; it does not revoke real credentials or disrupt the external provider.

## Native leaf retry groups

A frozen container, script or resource leaf with native `retryStrategy` can participate in new-parameter partial retry.
The author must still declare `retry-safe=idempotent` for changed calls and `result-reuse=same-inputs` for reused calls.
The impact checker identifies attempts through native Retry parent-child links, canonical attempt names, template identity and boundary identity.
It selects the affected retry group with all its attempts; Argo itself resets the selected group and creates new attempts with the changed inputs.
Unrelated completed retry groups retain their complete attempt history, including earlier failed attempts, original Pod UIDs, outputs and approvals.
A native retry dry-run must prove every retained fingerprint survives; otherwise the request is rejected before mutating the Workflow.
Retry groups around nested DAG/Steps, artifacts, memoized results, daemon tasks or ContainerSet remain outside this supported subset.
`retry_groups_e2e.py` covers actual failed and successful native attempts in both DAG and Steps, changed input, same-Workflow retry, retained history, new output and idempotent receipt recovery.

On current upstream main, native parameter override expands the reset to all children of an affected Steps/TaskGroup.
The checked extension passes an internal retry proof tied to the Workflow UID, resourceVersion and retained result fingerprints, so this conservative expansion can spare verified unrelated records.
Explicitly selected records are still reset; stale or mismatched proofs cannot retain them.
Ordinary native retries without that internal proof preserve upstream behavior, including loop re-expansion for parameter overrides.
The v4.0.1 compatibility path is checked separately because its native reset algorithm differs.

## Checked file reuse during new-parameter retry (v51)

Frozen native leaf templates can declare `parameters.workflows.example.io/artifact-checksums` as a JSON map from an output artifact name to its SHA-256 output parameter.
This contract is for non-sensitive files; it is not a secret-content comparison API.
The producer must already have a completed native result and the normal `result-reuse: same-inputs` declaration.
The current subset supports explicit HTTP locations with `archive.none`, raw input data or direct DAG/Steps artifact output arguments, and explicit producer dependencies.
Composite artifact forwarding, optional files, repository-default locations and other storage drivers remain unsupported for new-parameter retry.

Preview, retry application and parameterized Workflow admission read retained files through the native HTTP artifact driver with caller-scoped credential reads.
They require the original checksum output and identical storage content, with a total 64 MiB, 32-file and 10-second verification bound.
Errors omit URLs, credentials, content and checksum values.
Native retry retains the original producer status and Pod; the native executor downloads the retained file for the affected consumer.
No file data is copied into a parameter revision or retry receipt.
A file may still disappear after verification; that remains a native download failure, not proof of successful reuse or immutable object storage.

## Native credential lifecycle integration (v52)

The optional Vault Agent action is documented in [local testing](local-testing.md#native-dynamic-credential-leases).
Its full path verified real database lease renewal, revocation, denied acquisition, a fresh lease on native retry, preserved approval, reference-only history and TTL cleanup after completion or cancellation.
The API does not issue dynamic credentials itself; `perAttemptSecretLease` in its unsupported report describes the built-in server, not the ability to run an explicitly configured native credential provider.
Arbitrary HTTP and Plugin credential injection remains unsupported.

## Per-call result bindings

[Bound action results](execution-results.md) defines the experimental `ActionResultTemplate` resource and `WorkflowActionBinding` result bindings.
These freeze independently at each native call site.
Result reads use the originating attempt, including successful nodes retained across partial retry.
The server reads public outputs or an exact Workflow-owned result resource; separate execution adapters own external queries.
Refresh does not dispatch work or mutate Workflow state.
Existing artifact transport remains native.
