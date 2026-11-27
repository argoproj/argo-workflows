Description: Dynamic Resource Allocation resource claims for workflow pods
Authors: [Tsai Hsiu-Chi](https://github.com/thc1006)
Component: General
Issues: 16576

Workflow pods can now request devices through [Dynamic Resource Allocation](https://kubernetes.io/docs/concepts/scheduling-eviction/dynamic-resource-allocation/) via the new `resourceClaims` field, available at the workflow spec level and the template level.
Template-level `resourceClaims` replaces the workflow-level list as a whole rather than merging with it, and `podSpecPatch` still applies afterwards, merging over the entries by claim name.
A patch that moves a claim from one source to the other has to set the old source to `null`, otherwise the merge leaves both set and the API server rejects the pod.
Each entry names either an existing `ResourceClaim` or a `ResourceClaimTemplate` in the workflow's namespace, and a container asks for one by name through `resources.claims`.
Declaring `resourceClaims` on a template that creates no pod is rejected at submission: Steps, DAG and Suspend orchestrate other templates, and HTTP and Plugin run on the shared agent pod. The rest of a claim's shape, and whether a container's `resources.claims` names one the pod has, is left to the API server when the pod is created.
Previously the only way to attach a claim to a workflow pod was to hand-write the whole list in `podSpecPatch`.
Requires the `DynamicResourceAllocation` feature gate to be enabled on the cluster and a DRA driver to be installed.
Argo forwards the references and neither allocates devices nor manages claims: an existing `ResourceClaim` is created and deleted by whoever owns it, while one generated from a `ResourceClaimTemplate` is created for the pod and deleted with it.
