# Duration

Duration is a wrapper around time.Duration which supports correct marshaling to YAML and JSON. In particular, it marshals into strings, which can be used as map keys in json.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**duration** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.duration import Duration

# TODO update the JSON string below
json = "{}"
# create an instance of Duration from a JSON string
duration_instance = Duration.from_json(json)
# print the JSON string representation of the object
print(Duration.to_json())

# convert the object into a dict
duration_dict = duration_instance.to_dict()
# create an instance of Duration from a dict
duration_form_dict = duration.from_dict(duration_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


