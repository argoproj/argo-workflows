# Template Offloading

> v4.2 and after

Argo stores workflow templates inline in the Workflow object, so the total size of `spec.templates` is limited by the etcd size limit (1MB) — a limit that cannot be raised on managed Kubernetes services.
Template offloading stores `spec.templates` above a configurable size threshold in the persistence database, leaving only a small stub in the Workflow object.
This allows arbitrarily large DAGs (+1 MB) to run on clusters where the etcd limit cannot be changed (aws or any cloud with managed kubernetes).

To enable it, configure a Postgres, MySQL, or MariaDB database under `persistence` in [your configuration](workflow-controller-configmap.yaml) and set `templateOffLoad: true`:

```yaml
persistence: |
  templateOffLoad: true
```

Do not that you won't be able to submit workflow with `kubectl` only argo cli (needs to run without kubeconfig `KUBECONFIG=/dev/null argo submit workflow.json`) and REST api (`curl -X POST -d @payload.json http://<argo_url>/api/v1/workflows/argo`) will work.

On offloaded workflows you'll see

```yaml
Stored Template Specs:
  Hydrated: true
  UID: xyz
  Version: sha256:abc
```

Instead of `spec.templates`
