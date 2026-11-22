

# IoArgoprojWorkflowV1alpha1ExecutorPluginSidecar


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**automountServiceAccountToken** | **Boolean** | AutomountServiceAccountToken enables mounting the service account token. The service account must be named &lt;plugin-name&gt;-executor-plugin. Mutually exclusive with automountWorkflowServiceAccountToken. |  [optional]
**automountWorkflowServiceAccountToken** | **Boolean** | AutomountWorkflowServiceAccountToken mounts a projected token for the workflow&#39;s service account. Only applies to workflow-level plugins. Mutually exclusive with automountServiceAccountToken. |  [optional]
**container** | [**io.kubernetes.client.openapi.models.V1Container**](io.kubernetes.client.openapi.models.V1Container.md) |  | 



