# IoArgoprojEventsV1alpha1Template


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**affinity** | [**Affinity**](Affinity.md) |  | [optional] 
**container** | [**Container**](Container.md) |  | [optional] 
**image_pull_secrets** | [**[LocalObjectReference]**](LocalObjectReference.md) |  | [optional] 
**metadata** | [**IoArgoprojEventsV1alpha1Metadata**](IoArgoprojEventsV1alpha1Metadata.md) |  | [optional] 
**node_selector** | **{str: (str,)}** |  | [optional] 
**priority** | **int** |  | [optional] 
**priority_class_name** | **str** |  | [optional] 
**security_context** | [**PodSecurityContext**](PodSecurityContext.md) |  | [optional] 
**service_account_name** | **str** |  | [optional] 
**tolerations** | [**[Toleration]**](Toleration.md) |  | [optional] 
**volumes** | [**[Volume]**](Volume.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


