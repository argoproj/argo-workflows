Description: Add opt-in structured workflow parameters, forms, revisions and native action bindings
Authors: [jansuzhang](https://github.com/bjzhangshuai)
Component: General
Issues: 17221

An experimental namespaced API links typed parameter models, declarative forms and immutable saved revisions to native WorkflowTemplate inputs.
Argo Server and UI provide definition discovery, editing, version comparison, execution preview, submission and intermediate input.
Optional action bindings freeze native implementations and validate input/output contracts; a ResourceOperation reference adapter demonstrates recovery for one external operation.
Independent result templates bind public output or executor-maintained observations to each call, retaining original contracts for reused attempts and rendering declarative node views without business SDKs in Argo.
Reviewed parameter retry supports declared reusable results in the same Workflow while retaining unrelated successful calls and human decisions.
Unsupported graph, execution and artifact combinations are rejected before retry rather than silently resubmitting the workflow.
Secret references and optional Vault examples demonstrate version policies and native credential lifetimes without putting secret values into workflow arguments.
The feature is disabled by default, its CRDs and optional controllers are installed separately, and the example API group needs upstream design agreement.
See [the implementation decisions](docs/proposals/parameterization/implementation-decisions.md) and [local testing guide](docs/proposals/parameterization/local-testing.md) for the supported scope and runnable examples.
