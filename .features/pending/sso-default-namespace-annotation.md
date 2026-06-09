Description: Allow SSO ServiceAccount to override the UI default namespace per tenant
Author: [Robert Marklund](https://github.com/euforia)
Component: General
Issues: 16153

The SSO-mapping ServiceAccount can now declare the UI's default landing namespace via a new `workflows.argoproj.io/default-namespace` annotation.
When set, its value replaces `serviceAccountNamespace` in the claims returned by `/api/v1/userinfo`, so the UI lands the user in their tenant namespace instead of the install namespace.
This unblocks multi-tenant installs where SSO matching must happen in the install namespace but users only have permissions in a separate tenant namespace, and replaces the empty list and 403 errors first-time SSO users used to see.
The annotation is opt-in: absent annotation preserves prior behavior (the ServiceAccount's own namespace is used).
The annotation value is treated as a literal namespace name and does not expand any user's authorization — actual access is still governed by RoleBindings in the target namespace.
Three pre-existing UI bugs that prevented the new default from taking effect on first login are fixed alongside: `getCurrentNamespace()` no longer treats a persisted empty string as authoritative, `WorkflowsList` honors `userNamespace` from `/userinfo`, and `ClusterWorkflowTemplateDetails` no longer issues a cross-namespace list that 403s for non-cluster-admin tenants.
