Description: Strip the client-side-apply annotation from the controller's template caches to reduce memory usage
Authors: [Abhishek Sharma](https://github.com/abhisheksharma2411)
Component: General
Issues: 16566

The workflow-controller now removes the `kubectl.kubernetes.io/last-applied-configuration` annotation from WorkflowTemplates and ClusterWorkflowTemplates before storing them in its informer caches.
Client-side `kubectl apply`, and the GitOps tooling built on it, stores a copy of the whole spec in that annotation, which on a measured cluster was 3.6 KB of an 8.3 KB WorkflowTemplate.
Nothing reads it from the cache, so this reduces controller memory usage wherever templates are applied with client-side `kubectl apply`, with no functional change.
This applies only to the controller's template caches.
argo-server builds the same informers but serves those objects to clients, so its caches keep the annotation.
Objects stored in the cluster are unaffected.
