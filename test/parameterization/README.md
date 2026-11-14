# Testing structured parameters

Run the examples only against a dedicated local installation from [the local installation below](#install-an-isolated-example).
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

## Install an isolated example

Use a dedicated kubeconfig and context so the example does not replace an existing Argo installation or its cluster-scoped CRDs.
The example uses a separate k3d cluster running k3s, the namespace `argo-parameterization-lab` and controller instance ID `parameterization-lab`.
The bundled manifests use local images and server authentication for loopback access only.
Shared installations must configure client or SSO authentication and caller-scoped Kubernetes RBAC.

```sh
k3d cluster create argo-parameterization --image rancher/k3s:v1.32.8-k3s1 \
  --api-port 127.0.0.1:6551 --kubeconfig-update-default=false \
  --kubeconfig-switch-context=false \
  --k3s-arg '--disable=traefik,servicelb,metrics-server@server:0' \
  --k3s-arg '--secrets-encryption@server:0' \
  --k3s-arg '--secrets-encryption-provider=secretbox@server:0'
k3d kubeconfig get argo-parameterization > ./parameterization.kubeconfig
chmod 600 ./parameterization.kubeconfig
export KUBECONFIG="$PWD/parameterization.kubeconfig"
```

Build the UI, `dist/argo` with the `ui` build tag, and `dist/workflow-controller` for the architecture of the local cluster using this checkout's supported Go toolchain.
Build the image with `manifests/parameterization/Dockerfile.local`, then import it with `k3d image import IMAGE -c argo-parameterization`.
Set the image in `manifests/parameterization/local-install.yaml` to the imported tag.
The Deployment uses `imagePullPolicy: Never`.
Build the UI before the embedded server binary, and build the image after both binaries complete.

Install the native and experimental CRDs, immutable-snapshot protection and local namespace resources:

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

The Server requires `ARGO_PARAMETERIZATION_ENABLED=true`.
Use the enhanced Server and controller from the same checkout.
For clusters without ValidatingAdmissionPolicy, the separately installed namespace-scoped validating webhook also enforces immutable snapshots.
See `manifests/parameterization` for the optional admission and definition-status controller manifests.
Install admission protection before allowing users to submit parameterized Workflows.


## Result verification

Run the normal Go/UI parameterization tests and `test/parameterization/result_bindings_e2e.py` in the [isolated local installation](#install-an-isolated-example).
The case covers definition storage and validation, two calls to one action with different views, a native Resource query plus independent controller, read-only refresh, failed execution, changed-parameter retry on the same Workflow, preserved approval and successful node/Pod identities, frozen historical results, and the browser panel.
`action_calls_e2e.py`, `action_reference_e2e.py`, `definition_publications_e2e.py` and `execution_diff_e2e.py` cover the affected binding, original-retry, publication and diff paths.
No production cluster or external business provider is used.

## Synthetic dynamic credentials

Build `examples/parameterization/dynamic-credentials/Dockerfile` and import it into the isolated cluster.
Run `python3 test/parameterization/dynamic_leases_e2e.py --install-demo` to install its synthetic Vault/PostgreSQL backend and a 24-hour bootstrap reference.
Reinstall the fixture explicitly after the bootstrap token expires; never install it in a shared environment.

## Design and API notes

Design discussion belongs in [issue #17221](https://github.com/argoproj/argo-workflows/issues/17221).
The [prototype OpenAPI fixture](openapi.yaml) describes the experimental endpoints for development and testing.
The [user reference](../../docs/parameterization/contracts.md) lists installed resources and author annotations.
