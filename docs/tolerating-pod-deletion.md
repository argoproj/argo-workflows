# Tolerating Pod Deletion

> v2.12 and after

In Kubernetes, pods are cattle and can be deleted at any time. Deletion could be manually via `kubectl delete pod`, during a node drain, or for other reasons.

This can be very inconvenient, your workflow will error, but for reasons outside of your control.

A [pod disruption budget](https://raw.githubusercontent.com/argoproj/argo-workflows/main/examples/default-pdb-support.yaml) can reduce the likelihood of this happening. But, it cannot entirely prevent it.

To retry pods that were deleted, set `retryStrategy.retryPolicy: OnError`.

This can be set at a workflow-level, template-level, or globally (using [workflow defaults](default-workflow-specs.md))

## Example

Run the following workflow (which will sleep for 30s):

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: example
spec:
  retryStrategy:
   retryPolicy: OnError
   limit: 1
  entrypoint: main
  templates:
    - name: main
      container:
        image: docker/whalesay:latest
        command:
          - sleep
          - 30s
```

Then execute `kubectl delete pod example`. You'll see that the errored node is automatically retried.

💡 Read more on [architecting workflows for reliability](https://blog.argoproj.io/architecting-workflows-for-reliability-d33bd720c6cc).
