

# IoArgoprojWorkflowV1alpha1Pool

Pool runs a container or script template's tasks on a pool of long-lived worker pods, one task at a time per worker pod, instead of a pod per task.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**initialReplicas** | **Integer** | InitialReplicas is the number of worker pods a new pool starts with. It applies only when the pool is created. Defaults to 1. |  [optional]
**name** | **String** | Name of the pool. Templates with the same pool name and the same pod settings share a pool. Defaults to &#x60;&lt;workflowtemplate&gt;-&lt;template&gt;&#x60; for a template from a WorkflowTemplate or ClusterWorkflowTemplate, otherwise to the template&#39;s name. It must be a DNS label of at most 35 characters. It may hold a variable, such as &#x60;{{inputs.parameters.pool}}&#x60;, and is then checked once substituted. |  [optional]
**retentionSeconds** | **Integer** | RetentionSeconds is how long the pool is kept once no task uses it. The most recently dispatched task&#39;s value applies. Defaults to the controller&#39;s hot workers retention. |  [optional]



