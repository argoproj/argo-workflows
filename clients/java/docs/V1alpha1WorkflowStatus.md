

# V1alpha1WorkflowStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**compressedNodes** | **String** |  |  [optional]
**finishedAt** | [**V1Time**](V1Time.md) |  |  [optional]
**message** | **String** | A human readable message indicating details about why the workflow is in this condition. |  [optional]
**nodes** | [**Map&lt;String, Workflowv1alpha1NodeStatus&gt;**](Workflowv1alpha1NodeStatus.md) | Nodes is a mapping between a node ID and the node&#39;s status. |  [optional]
**offloadNodeStatusVersion** | **String** | Whether on not node status has been offloaded to a database. If exists, then Nodes and CompressedNodes will be empty. This will actually be populated with a hash of the offloaded data. |  [optional]
**outputs** | [**V1alpha1Outputs**](V1alpha1Outputs.md) |  |  [optional]
**persistentVolumeClaims** | [**List&lt;V1Volume&gt;**](V1Volume.md) | PersistentVolumeClaims tracks all PVCs that were created as part of the workflow. The contents of this list are drained at the end of the workflow. |  [optional]
**phase** | **String** | Phase a simple, high-level summary of where the workflow is in its lifecycle. |  [optional]
**startedAt** | [**V1Time**](V1Time.md) |  |  [optional]
**storedTemplates** | [**Map&lt;String, V1alpha1Template&gt;**](V1alpha1Template.md) | StoredTemplates is a mapping between a template ref and the node&#39;s status. |  [optional]



