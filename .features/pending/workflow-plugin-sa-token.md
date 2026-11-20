Description: Use rotating workflow service account tokens in workflow-level executor plugins
Authors: [ntny](https://github.com/ntny)
Component: General
Issues: 17247

Set `spec.executorPlugins[].spec.sidecar.automountWorkflowServiceAccountToken: true` to let a workflow-level executor plugin access the Kubernetes API with the workflow's service account and RBAC permissions.
The plugin and agent use short-lived projected tokens issued through the Kubernetes TokenRequest API and automatically rotated by kubelet, so neither needs a long-lived token Secret.
Plugins must reload the mounted token file when it changes.
This simplifies integrations with Kubeflow Pipelines by reusing the workflow's service account instead of provisioning a separate plugin account.
Only plugins declared in the Workflow spec can enable this option, keeping access to the workflow's identity under the workflow author's control.
The option defaults to `false` and is mutually exclusive with `automountServiceAccountToken`.
Existing plugin service account and ConfigMap token handling remains unchanged.
See "Service Account Tokens" in the Executor Plugins docs for configuration details.
