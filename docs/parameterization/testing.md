# Testing structured parameters

Run the examples only against a dedicated local installation from [getting started](getting-started.md).
The E2E cases use the Argo Server API, Kubernetes persistence, native controller/executor and synthetic Resource or container actions.
They retain execution records for inspection and bound their waits.

```sh
export KUBECONFIG="$PWD/parameterization.kubeconfig"
export ARGO_PARAMETERIZATION_CONTEXT=k3d-argo-parameterization
export ARGO_PARAMETERIZATION_URL=http://127.0.0.1:2747
export ARGO_CHECK_LOCAL_ASSETS=true

go test ./util/parameterization ./server/parameterization ./server/workflow ./workflow/util
(cd ui && yarn test src/parameterization --runInBand && yarn tsc --noEmit)
python3 test/parameterization/e2e.py
```

Browser cases require Python Playwright and Chromium.
Set `ARGO_PARAMETERIZATION_CHROMIUM` to an installed Chromium executable when Playwright does not bundle one.
`ARGO_CHECK_LOCAL_ASSETS=true` verifies that the served UI matches the completed local build before the main E2E creates a Workflow.

Select the suites for the behavior you changed rather than treating an unrelated local pass as full feature coverage.
All script names below are relative to `test/parameterization/`.

| Area | Full-path suites | What to inspect |
| --- | --- | --- |
| Save, form and execution | `e2e.py`, `schema_e2e.py`, `form_ui_e2e.py` | Saved values, compiled arguments, actual ConfigMap/container output and frozen form. |
| Definition lifecycle | `definitions_ui_e2e.py`, `definition_status_e2e.py`, `definition_publications_e2e.py` | Dry-run, conflicts, reference validation, immutable publication and explicit restore. |
| Original and changed retry | `partial_retry_e2e.py`, `nested_retry_e2e.py`, `retry_groups_e2e.py` | Stable Workflow UID, retained approval/Pod identities and changed output only for affected calls. |
| Retry conditions | `condition_retry_e2e.py` | Upstream output changes, consumer-only parameter changes, original/new retries and nested execute/skip transitions; one Workflow UID and retained independent approvals. |
| Loop topology | `topology_retry_e2e.py` | DAG/Steps expansions through two, three, zero and one items without positional result reuse. |
| Action references and contracts | `action_calls_e2e.py`, `action_reference_e2e.py`, `action_replacement_e2e.py` | Named call mapping, frozen source definitions and rejection of implementations that were not reviewed. |
| Artifacts | `artifact_calls_e2e.py`, `artifact_retry_e2e.py` | Real byte transfer, retained producer, tampering/missing-file rejection and downstream download. |
| Results | `result_bindings_e2e.py` | Different views for calls of the same action, adapter-owned observations, refresh and retained historical contracts. |
| Secret references | `secrets_e2e.py`, `sensitive_arrays_e2e.py`, `sensitive_inputs_e2e.py` | Reference-only history, native consumption, original retry and revoked-reference rejection. |
| External secrets | `external_secrets_e2e.py`, `secret_dispatch_e2e.py` | Version rotation, pinned retry, backend outage expiry and rejected dispatch after approval wait. |
| Dynamic leases | `dynamic_leases_e2e.py` | Fresh per-attempt lease, renewal, revocation, denied acquisition, cancellation and TTL cleanup. |
| Intermediate input | `inputs_e2e.py`, `input_options_e2e.py` | Frozen form, current allowed choices, response-loss recovery and downstream consumption. |
| History and diff | `paged_diff_e2e.py`, `paged_execution_diff_e2e.py`, `long_history_e2e.py` | Bounded pages, immutable comparison identity, concurrent saves and explicit cursor expiry. |
| Native references | `template_refs_e2e.py` | Nested DAG/Steps/cluster references, preview invalidation and source deletion after submission. |

The result observer, external-operation controller and Vault fixtures are examples with separate setup requirements documented beside their manifests.
They do not use production providers or account credentials.
A passing local run does not establish upstream CI status or community API approval.
