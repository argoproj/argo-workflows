# Artifact GC

> Since v3.4

Artifact GC allow you to specify a workflow whose output artifacts are deleted from their artifact repository (e.g. S3
bucket) on either completion or deletion of the workflow.

[Example](../examples/artifact-gc-workflow.yaml)

A strategy specified on the artifact take precedence over the strategy specified on the workflow spec. This allows you
to set a default policy for a workflow (e.g. all artifacts are deleted when the workflow completes), and override for
specific artifacts, e.g. never delete the most valuable output

Artifact GC happens after workflow are copied to the archive. Archived workflows will not contain information about GC.

If a workflow will need artifact GC, the controller adds a Kubernetes finalizer to it. This prevents the deletion of the
workflow until the finalizer is removed. When you delete a workflow (e.g. using `kubectl delete wf`) then the workflow
will not be removed from the system until all the artifacts are successfully deleted.

For each artifact that needs to be deleted, the controller will create a pod. It is possible more pods are created to
delete artifacts than are created to run the workflow. However, each pod has a strict security context and minimal
resource requests and limits.

The status of deletion is not reflected in the workflow graph. Instead, you can examine the workflow status conditions (
i.e. `status.conditions`) where the most recent error is reported.

## Troubleshooting

To determine why an artifact could not be deleted, examine the pod that was created to deleted it:

```bash
kubectl get pod -l workflow.argoproj.io/component=artifact-gc -l workflows.argoproj.io/workflow
```

If a pod failed to delete the artifact, then it will be `Failed`. Look at the pod logs to find out why.

If the artifact will never be deleted, because of some problem, remove the finalizer.

## Configuration

By default, we have assumed you do not want or need to run many garbage collection pods at once. That, unlike workflows,
you're happy for artifact GC to take a bit of time, with the benefit of preventing a stamped of pods being created,
potentially overloading your Kubernetes cluster. By default, the maximum concurrency (set
by `ARGO_ARTIFACT_MAX_CONCURRENT_PODS`) is `8`. 