# IoArgoprojWorkflowV1alpha1CronWorkflow

CronWorkflow is the definition of a scheduled workflow resource

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**metadata** | [**ObjectMeta**](ObjectMeta.md) |  | 
**spec** | [**IoArgoprojWorkflowV1alpha1CronWorkflowSpec**](IoArgoprojWorkflowV1alpha1CronWorkflowSpec.md) |  | 
**api_version** | **str** | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#resources | [optional] 
**kind** | **str** | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#types-kinds | [optional] 
**status** | [**IoArgoprojWorkflowV1alpha1CronWorkflowStatus**](IoArgoprojWorkflowV1alpha1CronWorkflowStatus.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


