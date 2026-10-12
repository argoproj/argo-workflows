# Optional ResourceOperation reference adapter

This example is an opt-in operator for one synthetic external operation. Argo's
native `resource` template creates/observes the CR and consumes its outputs. The
operator never schedules tasks or writes Workflow status. No production provider,
cloud credentials, delivery runtime or dashboard is used.

`workflow/resourceoperation` contains the provider-independent recovery logic.
Adapters implement `Store`, `Verifier` and `Provider`; the sample HTTP protocol is
not a universal provider API. `Store.SaveStatus` must use resourceVersion CAS.
Only the successful durable dispatch claim may call `Start`. Once dispatched,
lookup failures or missing receipts leave `OutcomeUnknown` and never grant
permission to start again. A crash between the durable claim and the external
request intentionally requires explicit recovery; this design does not claim
exactly-once effects against an arbitrary external system.

The sample uses an operator-configured endpoint and connection identity. Inputs
cannot select another URL. The invoker must match a Running native Resource Pod,
its Workflow UID, node name, materialized manifest, executor ServiceAccount, and
frozen submission action version. Initial identity checks happen before dispatch;
later observation uses the recorded snapshot digest. The example intentionally
supports a single-level invocation path. More general loop/nested invocation
identities need a separately verified adapter before being advertised.

The operation key stays stable on original-parameter native retry. Changing an
operation's inputs in place is forbidden. Explicitly creating a different external
operation requires a reviewed new logical intent and a new identity. The operation
CR is retained independently of Workflow cleanup and has no Workflow ownerReference.
Status is writable only by the operator; users can request cancellation through
`spec.control`, subject to Kubernetes RBAC.

`terminationPolicy: Detach` continues observation after Workflow termination;
`CancelIfSupported` sends one cancellation request and waits for the provider's
confirmation. Suspend does not cancel external work. Cancellation transport errors
do not imply cancellation success. Unsupported cancellation remains visible as a
condition. If an operator is absent, a resource task remains pending until its
native deadline; there is no automatic alternate scheduler.

Results must satisfy the frozen output schema before `status.phase: Succeeded`.
The native action contract separately validates extracted Argo output parameters.
Schema errors report a rejected contract without copying payloads into conditions.
The demo does not implement secret injection, arbitrary plugins, provider-specific
recovery, or a production connection registry.

## Local acceptance

Use only the dedicated `k3d-argo-parameterization` context. Build the Go example for
that node's architecture, package it with `Dockerfile`, and import the image into
that local k3s cluster. Install the ResourceOperation CRD from
`manifests/parameterization/crds/parameters.workflows.example.io_resourceoperations.yaml`.
The test creates its own namespaced adapter, synthetic HTTP service and RBAC:

```sh
ARGO_PARAMETERIZATION_URL=http://127.0.0.1:2747 \
ARGO_OPERATION_IMAGE=argo-operation-example:local-v2 \
python3 test/parameterization/operation_e2e.py
```

It exercises save/preview without operations, bound submission, native Resource
execution, lost HTTP responses and controller restart, native timeout followed by
original-parameter retry, terminate/cancel, detach, unknown outcomes, invalid
outputs, forged invoker rejection, immutable snapshots, and downstream output
consumption. An independent HTTP service counts actual creation requests. The
first timed-out task's external job is allowed to finish before the observation
retry, so the unchanged short native deadline is not mistaken for a guarantee
that an arbitrary external job will finish within it.

Completed native Workflow and ResourceOperation records remain available for
inspection. The test scales down only its own synthetic server/operator after
acceptance; set `ARGO_KEEP_OPERATION_DEMO=true` to leave them running for manual
experimentation. No production or prepublish resources are accessed.
