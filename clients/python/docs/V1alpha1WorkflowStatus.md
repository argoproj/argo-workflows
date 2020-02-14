# V1alpha1WorkflowStatus

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**compressed_nodes** | **str** |  | [optional] 
**finished_at** | [**V1Time**](V1Time.md) |  | [optional] 
**message** | **str** | A human readable message indicating details about why the workflow is in this condition. | [optional] 
**nodes** | [**dict(str, Workflowv1alpha1NodeStatus)**](Workflowv1alpha1NodeStatus.md) | Nodes is a mapping between a node ID and the node&#39;s status. | [optional] 
**offload_node_status_version** | **str** | Whether on not node status has been offloaded to a database. If exists, then Nodes and CompressedNodes will be empty. This will actually be populated with a hash of the offloaded data. | [optional] 
**outputs** | [**V1alpha1Outputs**](V1alpha1Outputs.md) |  | [optional] 
**persistent_volume_claims** | [**list[V1Volume]**](V1Volume.md) | PersistentVolumeClaims tracks all PVCs that were created as part of the workflow. The contents of this list are drained at the end of the workflow. | [optional] 
**phase** | **str** | Phase a simple, high-level summary of where the workflow is in its lifecycle. | [optional] 
**started_at** | [**V1Time**](V1Time.md) |  | [optional] 
**stored_templates** | [**dict(str, V1alpha1Template)**](V1alpha1Template.md) | StoredTemplates is a mapping between a template ref and the node&#39;s status. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


