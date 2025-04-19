# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EmailTrigger

EmailTrigger refers to the specification of the email notification trigger.

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**body** | **str** |  | [optional] 
**_from** | **str** |  | [optional] 
**host** | **str** | Host refers to the smtp host url to which email is send. | [optional] 
**parameters** | [**[GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter]**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter.md) |  | [optional] 
**port** | **int** |  | [optional] 
**smtp_password** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**subject** | **str** |  | [optional] 
**to** | **[str]** |  | [optional] 
**username** | **str** |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


