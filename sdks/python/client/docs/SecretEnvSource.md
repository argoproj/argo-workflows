# SecretEnvSource

SecretEnvSource selects a Secret to populate the environment variables with.  The contents of the target Secret's Data field will represent the key-value pairs as environment variables.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names | [optional] 
**optional** | **bool** | Specify whether the Secret must be defined | [optional] 

## Example

```python
from argo_workflows.models.secret_env_source import SecretEnvSource

# TODO update the JSON string below
json = "{}"
# create an instance of SecretEnvSource from a JSON string
secret_env_source_instance = SecretEnvSource.from_json(json)
# print the JSON string representation of the object
print(SecretEnvSource.to_json())

# convert the object into a dict
secret_env_source_dict = secret_env_source_instance.to_dict()
# create an instance of SecretEnvSource from a dict
secret_env_source_form_dict = secret_env_source.from_dict(secret_env_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


