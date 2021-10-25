# V1alpha1TemplateRef

TemplateRef is a reference of template resource.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**cluster_scope** | **bool** | ClusterScope indicates the referred template is cluster scoped (i.e. a ClusterWorkflowTemplate). | [optional] 
**name** | **str** | Name is the resource name of the template. | [optional] 
**runtime_resolution** | **bool** | RuntimeResolution skips validation at creation time. By enabling this option, you can create the referred workflow template before the actual runtime. DEPRECATED: This value is not used anymore and is ignored | [optional] 
**template** | **str** | Template is the name of referred template in the resource. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


