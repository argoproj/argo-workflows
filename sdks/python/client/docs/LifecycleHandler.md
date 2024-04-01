# LifecycleHandler

LifecycleHandler defines a specific action that should be taken in a lifecycle hook. One and only one of the fields, except TCPSocket must be specified.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**var_exec** | [**ExecAction**](ExecAction.md) |  | [optional] 
**http_get** | [**HTTPGetAction**](HTTPGetAction.md) |  | [optional] 
**tcp_socket** | [**TCPSocketAction**](TCPSocketAction.md) |  | [optional] 

## Example

```python
from argo_workflows.models.lifecycle_handler import LifecycleHandler

# TODO update the JSON string below
json = "{}"
# create an instance of LifecycleHandler from a JSON string
lifecycle_handler_instance = LifecycleHandler.from_json(json)
# print the JSON string representation of the object
print(LifecycleHandler.to_json())

# convert the object into a dict
lifecycle_handler_dict = lifecycle_handler_instance.to_dict()
# create an instance of LifecycleHandler from a dict
lifecycle_handler_form_dict = lifecycle_handler.from_dict(lifecycle_handler_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


