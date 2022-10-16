

# ObjectReference

ObjectReference contains enough information to let you inspect or modify the referred object.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**apiVersion** | **String** | API version of the referent. |  [optional]
**fieldPath** | **String** | If referring to a piece of an object instead of an entire object, this string should contain a valid JSON/Go field access statement, such as desiredState.manifest.containers[2]. For example, if the object reference is to a container within a pod, this would take on a value like: \&quot;spec.containers{name}\&quot; (where \&quot;name\&quot; refers to the name of the container that triggered the event) or if no container name is specified \&quot;spec.containers[2]\&quot; (container with index 2 in this pod). This syntax is chosen only to have some well-defined way of referencing a part of an object. |  [optional]
**kind** | **String** | Kind of the referent. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  [optional]
**name** | **String** | Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names |  [optional]
**namespace** | **String** | Namespace of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/ |  [optional]
**resourceVersion** | **String** | Specific resourceVersion to which this reference is made, if any. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency |  [optional]
**uid** | **String** | UID of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids |  [optional]



