# Lifecycle

Lifecycle describes actions that the management system should take in response to container lifecycle events. For the PostStart and PreStop lifecycle handlers, management of the container blocks until the action is complete, unless the container process fails, in which case the handler is aborted.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**post_start** | [**LifecycleHandler**](LifecycleHandler.md) |  | [optional] 
**pre_stop** | [**LifecycleHandler**](LifecycleHandler.md) |  | [optional] 

## Example

```python
from argo_workflows.models.lifecycle import Lifecycle

# TODO update the JSON string below
json = "{}"
# create an instance of Lifecycle from a JSON string
lifecycle_instance = Lifecycle.from_json(json)
# print the JSON string representation of the object
print(Lifecycle.to_json())

# convert the object into a dict
lifecycle_dict = lifecycle_instance.to_dict()
# create an instance of Lifecycle from a dict
lifecycle_form_dict = lifecycle.from_dict(lifecycle_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


