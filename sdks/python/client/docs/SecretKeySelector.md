# SecretKeySelector

SecretKeySelector selects a key of a Secret.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**key** | **str** | The key of the secret to select from.  Must be a valid secret key. | 
**name** | **str** | Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names | [optional] 
**optional** | **bool** | Specify whether the Secret or its key must be defined | [optional] 

## Example

```python
from argo_workflows.models.secret_key_selector import SecretKeySelector

# TODO update the JSON string below
json = "{}"
# create an instance of SecretKeySelector from a JSON string
secret_key_selector_instance = SecretKeySelector.from_json(json)
# print the JSON string representation of the object
print(SecretKeySelector.to_json())

# convert the object into a dict
secret_key_selector_dict = secret_key_selector_instance.to_dict()
# create an instance of SecretKeySelector from a dict
secret_key_selector_form_dict = secret_key_selector.from_dict(secret_key_selector_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


