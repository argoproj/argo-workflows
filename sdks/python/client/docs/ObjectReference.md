# ObjectReference

ObjectReference contains enough information to let you inspect or modify the referred object.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_version** | **str** | API version of the referent. | [optional] 
**field_path** | **str** | If referring to a piece of an object instead of an entire object, this string should contain a valid JSON/Go field access statement, such as desiredState.manifest.containers[2]. For example, if the object reference is to a container within a pod, this would take on a value like: \&quot;spec.containers{name}\&quot; (where \&quot;name\&quot; refers to the name of the container that triggered the event) or if no container name is specified \&quot;spec.containers[2]\&quot; (container with index 2 in this pod). This syntax is chosen only to have some well-defined way of referencing a part of an object. | [optional] 
**kind** | **str** | Kind of the referent. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds | [optional] 
**name** | **str** | Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names | [optional] 
**namespace** | **str** | Namespace of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/ | [optional] 
**resource_version** | **str** | Specific resourceVersion to which this reference is made, if any. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency | [optional] 
**uid** | **str** | UID of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids | [optional] 

## Example

```python
from argo_workflows.models.object_reference import ObjectReference

# TODO update the JSON string below
json = "{}"
# create an instance of ObjectReference from a JSON string
object_reference_instance = ObjectReference.from_json(json)
# print the JSON string representation of the object
print(ObjectReference.to_json())

# convert the object into a dict
object_reference_dict = object_reference_instance.to_dict()
# create an instance of ObjectReference from a dict
object_reference_form_dict = object_reference.from_dict(object_reference_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


