# PersistentVolumeClaim

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_version** | Option<**String**> | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources | [optional]
**kind** | Option<**String**> | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds | [optional]
**metadata** | Option<[**crate::models::ObjectMeta**](ObjectMeta.md)> |  | [optional]
**spec** | Option<[**crate::models::PersistentVolumeClaimSpec**](PersistentVolumeClaimSpec.md)> |  | [optional]
**status** | Option<[**crate::models::PersistentVolumeClaimStatus**](PersistentVolumeClaimStatus.md)> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


