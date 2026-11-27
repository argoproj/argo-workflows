# Resource Template

See [Kubernetes Resources](walk-through/kubernetes-resources.md).

## Executor recovery

Resource templates using `action: create` or `action: get` recover when their running executor pod is lost or fails because of an infrastructure condition such as eviction, OOM, or termination by a signal.
The controller creates a replacement executor to resume monitoring the resource.
Ordinary executor errors, such as an invalid manifest, still fail the node.

Recovery is always enabled for supported templates, with at most three recorded recoveries per node for failed or missing running executors.
The existing handling of missing pending pods can resubmit a pod without consuming a recovery attempt.
Recovery is independent of the controller's `failedPodRestart` setting, which handles pods whose main container never started.
Recorded attempts appear in the node's `resourcePodRecoveries` status field.

For `create`, the executor converts `metadata.generateName` to a deterministic name derived from the workflow UID and node ID, and adds annotations identifying the workflow node.
It stages a writable copy, leaving the original manifest artifact unchanged.
A replacement executor retrieves the existing resource if creation reports `AlreadyExists` and verifies those annotations before resuming; it refuses to adopt an unrelated object.
The first execution continues to report that error normally.
The same naming applies to inline manifests and manifests loaded from artifacts.
Client flags such as `--namespace` and `--context` apply to both creation and recovery; creation options such as `--validate` apply only to creation.
Recovery excludes `create` templates whose flags select additional files, Kustomize directories, raw API requests, interactive editing, or create subcommands, because those inputs cannot be safely replayed.

Other mutating resource actions are not automatically replayed after the executor has run: for example, repeating a JSON patch could append the same value twice.
