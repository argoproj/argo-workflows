# GoogleProtobufAny


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**type_url** | **str** |  | [optional] 
**value** | **bytearray** |  | [optional] 

## Example

```python
from argo_workflows.models.google_protobuf_any import GoogleProtobufAny

# TODO update the JSON string below
json = "{}"
# create an instance of GoogleProtobufAny from a JSON string
google_protobuf_any_instance = GoogleProtobufAny.from_json(json)
# print the JSON string representation of the object
print(GoogleProtobufAny.to_json())

# convert the object into a dict
google_protobuf_any_dict = google_protobuf_any_instance.to_dict()
# create an instance of GoogleProtobufAny from a dict
google_protobuf_any_form_dict = google_protobuf_any.from_dict(google_protobuf_any_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


