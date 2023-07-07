# IoArgoprojEventsV1alpha1Template

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**affinity** | Option<[**crate::models::Affinity**](Affinity.md)> |  | [optional]
**container** | Option<[**crate::models::Container**](Container.md)> |  | [optional]
**image_pull_secrets** | Option<[**Vec<crate::models::LocalObjectReference>**](LocalObjectReference.md)> |  | [optional]
**metadata** | Option<[**crate::models::IoArgoprojEventsV1alpha1Metadata**](io.argoproj.events.v1alpha1.Metadata.md)> |  | [optional]
**node_selector** | Option<**::std::collections::HashMap<String, String>**> |  | [optional]
**priority** | Option<**i32**> |  | [optional]
**priority_class_name** | Option<**String**> |  | [optional]
**security_context** | Option<[**crate::models::PodSecurityContext**](PodSecurityContext.md)> |  | [optional]
**service_account_name** | Option<**String**> |  | [optional]
**tolerations** | Option<[**Vec<crate::models::Toleration>**](Toleration.md)> |  | [optional]
**volumes** | Option<[**Vec<crate::models::Volume>**](Volume.md)> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


