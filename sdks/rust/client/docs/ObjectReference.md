# ObjectReference

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_version** | Option<**String**> | API version of the referent. | [optional]
**field_path** | Option<**String**> | If referring to a piece of an object instead of an entire object, this string should contain a valid JSON/Go field access statement, such as desiredState.manifest.containers[2]. For example, if the object reference is to a container within a pod, this would take on a value like: \"spec.containers{name}\" (where \"name\" refers to the name of the container that triggered the event) or if no container name is specified \"spec.containers[2]\" (container with index 2 in this pod). This syntax is chosen only to have some well-defined way of referencing a part of an object. | [optional]
**kind** | Option<**String**> | Kind of the referent. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds | [optional]
**name** | Option<**String**> | Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names | [optional]
**namespace** | Option<**String**> | Namespace of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/ | [optional]
**resource_version** | Option<**String**> | Specific resourceVersion to which this reference is made, if any. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency | [optional]
**uid** | Option<**String**> | UID of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


