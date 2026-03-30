# Argo Workflows — Aggregated API Server

An optional component that registers a [Kubernetes aggregated API server](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/apiserver-aggregation/) for the `argoproj.io/v1alpha1` group. Resources are persisted in **PostgreSQL** or **SQLite** instead of etcd, enabling long-term retention, SQL querying, and horizontal scalability.

## Table of Contents

- [Why](#why)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Kustomize Overlays](#kustomize-overlays)
  - [PostgreSQL (recommended)](#postgresql-recommended)
  - [SQLite (dev / single-node)](#sqlite-dev--single-node)
- [Configuration Reference](#configuration-reference)
- [Database Schema](#database-schema)
- [How Watch Works](#how-watch-works)
- [Troubleshooting](#troubleshooting)
- [Known Limitations](#known-limitations)

---

## Why

| | CRD + etcd (default) | Aggregated API Server + SQL |
|---|---|---|
| Storage | etcd (≤1.5 MB/object) | PostgreSQL or SQLite — no practical limit |
| Long-term retention | etcd fills up; requires pruning | Retain indefinitely |
| Querying | Label/field selectors only | Full SQL — analytics, ad-hoc queries |
| HA | etcd cluster required | PostgreSQL replication / managed RDS |
| Backup | etcd snapshots | Standard `pg_dump` / cloud snapshots |

---

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                  Kubernetes API Server                    │
│                                                          │
│  GET /apis/argoproj.io/v1alpha1/...                      │
│        │                                                 │
│        │  APIService v1alpha1.argoproj.io                │
│        ▼                                                 │
│  Proxy → argo-aggregated-apiserver:6443                  │
└──────────────────────────────────────────────────────────┘
                         │
                         ▼
         ┌───────────────────────────────┐
         │  SimpleAggregatedServer       │
         │  pkg/apiserver/simpleserver   │
         │                               │
         │  Routes:                      │
         │  /apis/.../namespaces/{ns}/   │
         │    {resource}[/{name}]        │
         │  /apis/.../{resource}         │  ← cluster-scoped
         │  /apis  (discovery)           │
         │  /healthz                     │
         └───────────┬───────────────────┘
                     │
                     ▼
         ┌───────────────────────────────┐
         │  GenericStore (GORM)          │
         │  pkg/storage/rest/            │
         │                               │
         │  Get / List / Create /        │
         │  Update / Delete / Watch      │
         └───────────┬───────────────────┘
                     │
           ┌─────────┴─────────┐
           ▼                   ▼
    ┌─────────────┐    ┌──────────────┐
    │  PostgreSQL │    │   SQLite     │
    │  (default)  │    │  (dev only)  │
    └─────────────┘    └──────────────┘
```

**Resource types served** (all `argoproj.io/v1alpha1`):

| Resource | Namespaced | E2E validated |
|---|---|---|
| `workflows` | ✓ | ✓ |
| `workflowtemplates` | ✓ | ✓ |
| `clusterworkflowtemplates` | | ✓ |
| `cronworkflows` | ✓ | |
| `workflowtasksets` | ✓ | ✓ (internal, used by workflow executor) |
| `workflowtaskresults` | ✓ | ✓ (internal, used by workflow executor) |
| `workflowartifactgctasks` | ✓ | |
| `workfloweventbindings` | ✓ | |

### Startup Order

The aggregated server runs as a **separate Deployment** (`argo-aggregated-apiserver`). It must be ready before the workflow-controller starts, otherwise the controller's initial list/watch will fail. The readiness probe on `/apis` ensures the kube-apiserver only routes traffic once the server is healthy.

If you restart both at the same time, roll the controller after the aggregated server is `1/1 Ready`:

```bash
kubectl rollout status deployment/argo-aggregated-apiserver -n argo
kubectl rollout restart deployment/workflow-controller -n argo
```

---

## Quick Start

> Prerequisites: Kubernetes 1.25+, Argo Workflows installed in the `argo` namespace, `kubectl` with `kustomize` support.

> **⚠️ Important:** The `argoproj.io/v1alpha1` resource group must not have any CRDs installed. If you installed Argo from `manifests/quick-start/minimal` (or any install that includes CRDs), Kubernetes automatically creates a *local* APIService for that group which takes priority over the aggregated server. Delete any installed `argoproj.io` CRDs first:
>
> ```bash
> kubectl delete crd \
>   workflowartifactgctasks.argoproj.io \
>   workfloweventbindings.argoproj.io \
>   workflowtaskresults.argoproj.io \
>   workflowtasksets.argoproj.io \
>   workflows.argoproj.io \
>   workflowtemplates.argoproj.io \
>   clusterworkflowtemplates.argoproj.io \
>   cronworkflows.argoproj.io 2>/dev/null || true
> ```
>
> The aggregated server handles all `argoproj.io/v1alpha1` resources; no CRDs are needed.

```bash
# Deploy with PostgreSQL (recommended)
kubectl apply -k manifests/overlays/aggregated-apiserver/postgres

# Verify the APIService is Available
kubectl get apiservice v1alpha1.argoproj.io

# Submit a workflow and confirm it goes through the SQL backend
kubectl create -f examples/hello-world.yaml -n argo
kubectl get workflows -n argo

# Create and reference a ClusterWorkflowTemplate
kubectl apply -f - <<'EOF'
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: my-template
spec:
  templates:
  - name: hello
    container:
      image: alpine:3.18
      command: [echo, hello]
EOF
kubectl get clusterworkflowtemplates
```

---

## Kustomize Overlays

```
manifests/
├── base/aggregated-apiserver/          # Base resources (deployment, service, RBAC, APIService)
├── components/aggregated-apiserver/    # Optional: patches argo-server to enable aggregated API
└── overlays/aggregated-apiserver/
    ├── postgres/                       # PostgreSQL-backed (recommended)
    └── sqlite/                         # SQLite-backed (dev/single-node)
```

### PostgreSQL (recommended)

**Path:** `manifests/overlays/aggregated-apiserver/postgres/`

What it deploys:
- `argo-aggregated-apiserver` Deployment — server process pointing at postgres
- `aggregated-apiserver-postgres-dsn` Secret — DSN for the shared postgres pod
- Reuses the postgres pod from `components/postgres` (same pod used for workflow archiving)
- `APIService v1alpha1.argoproj.io` → routes all `argoproj.io/v1alpha1` requests here

```bash
kubectl apply -k manifests/overlays/aggregated-apiserver/postgres
```

**External database** — edit the secret before applying:

```yaml
# manifests/overlays/aggregated-apiserver/postgres/aggregated-apiserver-postgres-secret.yaml
stringData:
  dsn: "postgresql://myuser:mypassword@my-rds-host:5432/argo?sslmode=require"
```

Or patch it post-deploy:

```bash
kubectl create secret generic aggregated-apiserver-postgres-dsn \
  --from-literal=dsn="postgresql://myuser:mypassword@my-rds.example.com:5432/argo?sslmode=require" \
  -n argo --dry-run=client -o yaml | kubectl apply -f -
kubectl rollout restart deployment/argo-aggregated-apiserver -n argo
```

### SQLite (dev / single-node)

**Path:** `manifests/overlays/aggregated-apiserver/sqlite/`

What it deploys:
- `argo-aggregated-apiserver` Deployment — single replica
- `aggregated-apiserver-sqlite` PersistentVolumeClaim — 5 Gi for the database file
- `APIService v1alpha1.argoproj.io`

```bash
kubectl apply -k manifests/overlays/aggregated-apiserver/sqlite
```

> **⚠️ Warning:** SQLite requires exactly **1 replica**. Multiple writers to the same file will corrupt the database. For any HA setup, use PostgreSQL.

To adjust the PVC size, edit `sqlite-pvc.yaml` before applying:

```yaml
spec:
  resources:
    requests:
      storage: 20Gi
```

---

## Configuration Reference

### CLI Flags

| Flag | Default | Description |
|---|---|---|
| `--enable-aggregated-apiserver` | `false` | Enable the SQL-backed aggregated API server |
| `--db-driver` | `sqlite` | Database driver: `sqlite` or `postgres` |
| `--db-dsn` | `argo.db` | Database connection string or file path |
| `--aggregated-api-port` | `6443` | TLS port the aggregated server listens on |

### Environment Variables

All flags can be set via environment variables:

| Environment Variable | Equivalent Flag |
|---|---|
| `ARGO_DB_DRIVER=postgres` | `--db-driver` |
| `ARGO_DB_DSN=postgresql://...` | `--db-dsn` |
| `ARGO_AGGREGATED_API_PORT=6443` | `--aggregated-api-port` |

The manifests inject `ARGO_DB_DSN` from the `postgres-credentials` Secret so the DSN is not exposed in pod command-line arguments.

### DSN Examples

```bash
# SQLite — file path (use with sqlite overlay)
--db-dsn /data/argo.db

# SQLite — in-memory (testing only; lost on pod restart)
--db-dsn :memory:

# PostgreSQL — in-cluster (bundled postgres pod, default credentials)
--db-dsn "postgresql://argo:argo@postgres:5432/argo?sslmode=disable"

# PostgreSQL — external with TLS
--db-dsn "postgresql://argo:argo@my-db.example.com:5432/argo?sslmode=require"
```

---

## Database Schema

Four tables are created automatically on first start via GORM AutoMigrate:

### `resource_records`

Stores every Argo resource as a JSON blob.

| Column | Type | Description |
|---|---|---|
| `id` | int | Primary key |
| `kind` | string | e.g. `Workflow`, `WorkflowTemplate` |
| `namespace` | string | Kubernetes namespace (empty for cluster-scoped) |
| `name` | string | Resource name |
| `uid` | string | UUID, unique per resource |
| `resource_version` | int64 | Monotonically increasing (global counter) |
| `generation` | int64 | Spec change counter |
| `data` | text | Full JSON serialisation |
| `created_at` | timestamp | |
| `updated_at` | timestamp | |
| `deleted_at` | timestamp | Soft delete |

### `resource_labels`

Enables efficient label selector queries without scanning `data`.

| Column | Type | Description |
|---|---|---|
| `resource_id` | int | FK → resource_records |
| `key` | string | Label key |
| `value` | string | Label value |

### `resource_version_counter`

Single-row table providing a global monotonic counter shared across all resource types.

### `watch_events`

Stores recent events for watch reconnection replay (5-minute TTL).

---

## How Watch Works

The server implements the full Kubernetes watch protocol including the WatchList feature (k8s 1.27+):

1. **Initial events** — client sends `?watch=true&sendInitialEvents=true&resourceVersionMatch=NotOlderThan`
2. Server sends all existing objects as `ADDED` events
3. Server sends a `BOOKMARK` with annotation `k8s.io/initial-events-end: "true"`
4. Server streams subsequent change events as they arrive
5. Server sends periodic `BOOKMARK` events every 30 s for reconnection support

**Wire format** — the watch stream serialises events as newline-delimited JSON objects with lowercase field names (`type`, `object`) to match the Kubernetes API convention. BOOKMARK events carry the same `kind` as the resource being watched.

---

## Troubleshooting

### Pod crashes with `strconv.ParseInt: parsing "tcp://..."` on startup

Kubernetes automatically injects service environment variables for every Service in the namespace. If a Service named `argo-aggregated-apiserver` exists, k8s injects `ARGO_AGGREGATED_APISERVER_PORT=tcp://x.x.x.x:6443` which collides with viper's env binding for the old `--aggregated-apiserver-port` flag.

This was fixed by renaming the flag to `--aggregated-api-port` (env: `ARGO_AGGREGATED_API_PORT`). If you see this with an older build, rebuild with the renamed flag or explicitly pass `--aggregated-api-port 6443` in the deployment args.

### Controller not picking up new workflows after redeployment

If you redeploy the aggregated server (e.g. change the image), the controller's watch connection may be stale. Restart it:

```bash
kubectl rollout status deployment/argo-aggregated-apiserver -n argo
kubectl rollout restart deployment/workflow-controller -n argo
```

### `the server could not find the requested resource`

The `APIService` is either not registered, auto-managed as local (due to installed CRDs), or the server is unreachable.

**Step 1** — check if the APIService points to the service (not local):

```bash
kubectl get apiservice v1alpha1.argoproj.io -o jsonpath='{.spec.service}'
# Should return: {"name":"argo-aggregated-apiserver","namespace":"argo","port":6443}
# If empty, the APIService is local (CRD-backed) — see below
```

**If the APIService is local** — installed CRDs are taking precedence. Delete them and re-apply:

```bash
kubectl delete crd \
  workflowartifactgctasks.argoproj.io workfloweventbindings.argoproj.io \
  workflowtaskresults.argoproj.io workflowtasksets.argoproj.io \
  workflows.argoproj.io workflowtemplates.argoproj.io \
  clusterworkflowtemplates.argoproj.io cronworkflows.argoproj.io 2>/dev/null || true
kubectl apply -k manifests/overlays/aggregated-apiserver/postgres
```

**Step 2** — if the service is correctly set but still failing:

```bash
kubectl get apiservice v1alpha1.argoproj.io
# Should show: AVAILABLE True

kubectl get pod -n argo -l app=argo-aggregated-apiserver
kubectl logs -n argo -l app=argo-aggregated-apiserver | grep -i "error\|ready"
```

### APIService shows `False / ServiceUnavailable`

The kube-apiserver cannot reach the aggregated server. Check:

1. Pod is running: `kubectl get pod -n argo -l app=argo-aggregated-apiserver`
2. Service exists: `kubectl get svc argo-aggregated-apiserver -n argo`
3. Readiness probe passes: look for `serving` in the pod logs

### PostgreSQL connection refused

The aggregated server pod starts before postgres is ready. It retries automatically. Force a fresh start:

```bash
kubectl rollout restart deployment/argo-aggregated-apiserver -n argo
```

### `Warning: event bookmark expired` in workflow-controller logs

Non-fatal. The controller's watch reconnection timer fired before a periodic BOOKMARK arrived. The controller re-lists and re-watches automatically; workflow execution is unaffected.

### `Unexpected watch event object type` warning in controller logs

Non-fatal cosmetic warning. Occurs every 30 s when the BOOKMARK ticker fires for secondary resource types. Workflow execution continues normally.

### SQLite `database is locked`

More than one replica is running with the SQLite overlay. Scale down:

```bash
kubectl scale deployment argo-aggregated-apiserver --replicas=1 -n argo
```

Or migrate to the postgres overlay.

### List/get returns objects without `kind`/`apiVersion`

You are running an older build. The `setTypeMeta` fix in `pkg/storage/rest/generic_store.go` ensures TypeMeta is populated on all responses. Rebuild and redeploy the image.

---

## Known Limitations

- **No CRDs allowed** — all `argoproj.io/v1alpha1` CRDs must be absent from the cluster. Kubernetes automatically creates a *local* APIService (backed by etcd) when any CRD for the group is present, which takes precedence over the aggregated server. The aggregated server itself serves all resource types (including `workflowtaskresults`, `workflowtasksets`, etc.) so CRDs are not needed.
- **SQLite is single-writer** — do not run more than 1 replica with the SQLite overlay. Use PostgreSQL for HA.
- **`argoproj.io/v1alpha1` only** — all other resource types continue to flow through etcd. This is not a full etcd replacement.
- **Self-signed TLS** — certificates are generated at startup. Restart the pod to regenerate. For production, provide a proper CA and certificate via a Secret and mount it into the pod.
- **No server-side apply** — `kubectl apply --server-side` is not supported; use standard `kubectl apply`.
- **No admission webhooks** — Argo's mutating/validating webhooks do not run in the aggregated path. Resource validation is best-effort.
- **`--auth-mode server` only** — other auth modes are not tested with the aggregated server path.
