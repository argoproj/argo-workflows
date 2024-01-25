

# IoArgoprojWorkflowV1alpha1WorkflowStatus

WorkflowStatus contains overall status information about a workflow

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artifactGCStatus** | [**IoArgoprojWorkflowV1alpha1ArtGCStatus**](IoArgoprojWorkflowV1alpha1ArtGCStatus.md) |  |  [optional]
**artifactRepositoryRef** | [**IoArgoprojWorkflowV1alpha1ArtifactRepositoryRefStatus**](IoArgoprojWorkflowV1alpha1ArtifactRepositoryRefStatus.md) |  |  [optional]
**compressedNodes** | **String** | Compressed and base64 decoded Nodes map |  [optional]
**conditions** | [**List&lt;IoArgoprojWorkflowV1alpha1Condition&gt;**](IoArgoprojWorkflowV1alpha1Condition.md) | Conditions is a list of conditions the Workflow may have |  [optional]
**estimatedDuration** | **Integer** | EstimatedDuration in seconds. |  [optional]
**finishedAt** | **java.time.Instant** |  |  [optional]
**message** | **String** | A human readable message indicating details about why the workflow is in this condition. |  [optional]
**nodes** | [**Map&lt;String, IoArgoprojWorkflowV1alpha1NodeStatus&gt;**](IoArgoprojWorkflowV1alpha1NodeStatus.md) | Nodes is a mapping between a node ID and the node&#39;s status. |  [optional]
**offloadNodeStatusVersion** | **String** | Whether on not node status has been offloaded to a database. If exists, then Nodes and CompressedNodes will be empty. This will actually be populated with a hash of the offloaded data. |  [optional]
**outputs** | [**IoArgoprojWorkflowV1alpha1Outputs**](IoArgoprojWorkflowV1alpha1Outputs.md) |  |  [optional]
**persistentVolumeClaims** | [**List&lt;io.kubernetes.client.openapi.models.V1Volume&gt;**](io.kubernetes.client.openapi.models.V1Volume.md) | PersistentVolumeClaims tracks all PVCs that were created as part of the io.argoproj.workflow.v1alpha1. The contents of this list are drained at the end of the workflow. |  [optional]
**phase** | **String** | Phase a simple, high-level summary of where the workflow is in its lifecycle. Will be \&quot;\&quot; (Unknown), \&quot;Pending\&quot;, or \&quot;Running\&quot; before the workflow is completed, and \&quot;Succeeded\&quot;, \&quot;Failed\&quot; or \&quot;Error\&quot; once the workflow has completed. |  [optional]
**progress** | **String** | Progress to completion |  [optional]
**resourcesDuration** | **Map&lt;String, Long&gt;** | ResourcesDuration is the total for the workflow |  [optional]
**startedAt** | **java.time.Instant** |  |  [optional]
**storedTemplates** | [**Map&lt;String, IoArgoprojWorkflowV1alpha1Template&gt;**](IoArgoprojWorkflowV1alpha1Template.md) | StoredTemplates is a mapping between a template ref and the node&#39;s status. |  [optional]
**storedWorkflowTemplateSpec** | [**IoArgoprojWorkflowV1alpha1WorkflowSpec**](IoArgoprojWorkflowV1alpha1WorkflowSpec.md) |  |  [optional]
**synchronization** | [**IoArgoprojWorkflowV1alpha1SynchronizationStatus**](IoArgoprojWorkflowV1alpha1SynchronizationStatus.md) |  |  [optional]
**taskResultsCompletionStatus** | **Map&lt;String, Boolean&gt;** | TaskResultsCompletionStatus tracks task result completion status (mapped by node ID). Used to prevent premature archiving and garbage collection. |  [optional]



