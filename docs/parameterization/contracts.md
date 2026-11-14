# Structured parameter reference

Structured parameters use the experimental API group `parameters.workflows.example.io/v1alpha1`.
The feature is disabled by default; see [installation and permissions](getting-started.md#enable-structured-parameters).
The definitions compile into native Workflows, which the Argo controller schedules.

## Resources and references

| Resource | Purpose and references |
| --- | --- |
| ParameterTemplate | Defines the value schema, defaults and identity rules. |
| WorkflowForm | Refers to a parameter model and describes its fields, layout, conditions and option sources. |
| WorkflowParameterBinding | Maps a parameter model into a native WorkflowTemplate or ClusterWorkflowTemplate; it can select a form and an action binding. |
| ActionDefinition | Defines the input/output contract and a native implementation, either inline or by reference. |
| WorkflowActionBinding | Selects actions at named call paths and maps their parameters, artifacts and result contracts. |
| ActionResultTemplate | Describes a result source, public schema and declarative views; each call selects its own result bindings. |
| ParameterSet | Holds the current saved-revision reference; concurrent saves use Kubernetes resourceVersion preconditions. |
| ParameterRevision | Stores immutable values and their parameter model snapshot. |
| WorkflowSubmission | Freezes the resolved definitions and native execution for one explicit submission. |
| ParameterRetry | Records the previous execution snapshot and the reviewed local retry, preserving the Workflow UID. |
| ParameterNodeInput | Records accepted intermediate form input for a native suspend node. |
| ResourceOperation | Optionally records one external operation; an execution adapter owns observation and recovery. |

The API types in `pkg/apis/parameterization/v1alpha1` and the generated schemas in `manifests/parameterization/crds.yaml` define the fields.
A saved revision freezes values and their model; a submission separately freezes the resolved definitions and native templates.
Saving uses Kubernetes `resourceVersion` preconditions and rejects stale edits.
The value schema uses JSON Schema draft-07; unsupported keywords and remote references are rejected.
Explicit false, zero, null and empty values survive defaulting.
Use Secret references for sensitive values, not plaintext Workflow arguments.

## Template annotations

These annotations use the prefix `parameters.workflows.example.io/`.
Set them on the native template that owns the declared behavior.
Declarations do not grant Kubernetes permissions or make external operations idempotent.

| Annotation | Value and effect |
| --- | --- |
| `bindable-calls` | JSON array of named child calls that a WorkflowActionBinding may bind, such as `["build"]`. |
| `retry-safe` | `idempotent` permits an affected executable leaf to run again after the server checks its retry plan. |
| `result-reuse` | `same-inputs` permits the server to retain a result only when its execution fingerprint still matches. |
| `retry-dependencies` | `inputs` declares ordering-only DAG dependencies; predecessor output references and changed arguments still invalidate the consumer. |
| `retry-loop-calls` | JSON array of named loops allowed to replace their whole expansion; changed identities do not reuse positional results. |
| `artifact-checksums` | JSON map from an output artifact name to its SHA-256 output parameter; enables checked reuse of supported HTTP artifacts. |
| `action-replacement-policy` | JSON object with `contract`, `interface` and `implementationDigests`; restricts replacement to an author-reviewed action interface and implementations. |

Do not declare ordering-only dependencies when an operation depends on predecessor side effects that its inputs do not describe.

## Execution annotations

The server maintains these annotations; copying or editing them does not create a valid execution receipt.
They also use the prefix `parameters.workflows.example.io/`.

| Annotation | Meaning |
| --- | --- |
| `submission` | Name of the immutable WorkflowSubmission that proves the initial execution contract. |
| `revision` | ParameterRevision used by the current execution attempt. |
| `execution-digest` | Digest checked against the resolved submission contract. |
| `retry` | ParameterRetry receipt for the current checked retry. |
| `definition-*` | Publication metadata recording source identity, content digest, family, publisher and publication time. |

## Checked new-parameter native retry

Preview a retry before applying it.
The server checks the affected calls, retained input/output fingerprints and the current Workflow identity and version.
A concurrent change invalidates the preview.
Successful unaffected calls and approval decisions remain retained; an unsupported plan fails before mutation.
Saving parameters or refreshing results does not start a retry.

Selected `callPaths` are relative to the frozen entrypoint.
Omitting `revision` uses the original inputs; supplying a revision also includes calls affected by changed values.
A replacement binding may change wholly unvisited implementations, but the root graph, execution scope and visited templates remain frozen.
Arbitrary topology replacement, memoized results, composite retry groups and unsupported artifact storage are rejected.
The retry keeps the Workflow UID and records the previous attempt in ParameterRetry.

## Registered external Secret backend

The optional `parameterization-secret-controller` reads administrator-configured Vault KV-v2 versions into immutable Kubernetes Secrets.
Configure encryption at rest and metadata-only Secret auditing before enabling it.
The backend origin, token file, paths and destination belong to the administrator configuration, not the parameter form.
Original retries keep their pinned Secret version; a new submission or reviewed retry can resolve a newer version.

The secret controller renews a 60-second verification window every 15 seconds.
Preview, submission, intermediate input, retry and new task Pod dispatch reject expired verification metadata.
Existing Pods and successful nodes remain unchanged, and a Pod created before expiry may start later.
The secret controller cannot revoke credentials already copied into a running process.
See [sensitive values](getting-started.md#sensitive-values) and [bound action results](execution-results.md) for the user-facing flows.
