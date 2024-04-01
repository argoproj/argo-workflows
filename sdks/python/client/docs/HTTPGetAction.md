# HTTPGetAction

HTTPGetAction describes an action based on HTTP Get requests.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**host** | **str** | Host name to connect to, defaults to the pod IP. You probably want to set \&quot;Host\&quot; in httpHeaders instead. | [optional] 
**http_headers** | [**List[HTTPHeader]**](HTTPHeader.md) | Custom headers to set in the request. HTTP allows repeated headers. | [optional] 
**path** | **str** | Path to access on the HTTP server. | [optional] 
**port** | **str** |  | 
**scheme** | **str** | Scheme to use for connecting to the host. Defaults to HTTP.  Possible enum values:  - &#x60;\&quot;HTTP\&quot;&#x60; means that the scheme used will be http://  - &#x60;\&quot;HTTPS\&quot;&#x60; means that the scheme used will be https:// | [optional] 

## Example

```python
from argo_workflows.models.http_get_action import HTTPGetAction

# TODO update the JSON string below
json = "{}"
# create an instance of HTTPGetAction from a JSON string
http_get_action_instance = HTTPGetAction.from_json(json)
# print the JSON string representation of the object
print(HTTPGetAction.to_json())

# convert the object into a dict
http_get_action_dict = http_get_action_instance.to_dict()
# create an instance of HTTPGetAction from a dict
http_get_action_form_dict = http_get_action.from_dict(http_get_action_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


