# IoArgoprojEventsV1alpha1Template

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**affinity** | [**Affinity**](Affinity.md) |  | [optional] 
**container** | [**Container**](Container.md) |  | [optional] 
**image_pull_secrets** | [**list[LocalObjectReference]**](LocalObjectReference.md) |  | [optional] 
**metadata** | [**IoArgoprojEventsV1alpha1Metadata**](IoArgoprojEventsV1alpha1Metadata.md) |  | [optional] 
**node_selector** | **dict(str, str)** |  | [optional] 
**priority** | **int** |  | [optional] 
**priority_class_name** | **str** |  | [optional] 
**security_context** | [**PodSecurityContext**](PodSecurityContext.md) |  | [optional] 
**service_account_name** | **str** |  | [optional] 
**tolerations** | [**list[Toleration]**](Toleration.md) |  | [optional] 
**volumes** | [**list[Volume]**](Volume.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


