# SecretProjection

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | Option<[**Vec<crate::models::KeyToPath>**](KeyToPath.md)> | If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'. | [optional]
**name** | Option<**String**> | Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names | [optional]
**optional** | Option<**bool**> | Specify whether the Secret or its key must be defined | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


