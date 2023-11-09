# HTTPGetAction

HTTPGetAction describes an action based on HTTP Get requests.

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**port** | **str** |  | 
**host** | **str** | Host name to connect to, defaults to the pod IP. You probably want to set \&quot;Host\&quot; in httpHeaders instead. | [optional] 
**http_headers** | [**[HTTPHeader]**](HTTPHeader.md) | Custom headers to set in the request. HTTP allows repeated headers. | [optional] 
**path** | **str** | Path to access on the HTTP server. | [optional] 
**scheme** | **str** | Scheme to use for connecting to the host. Defaults to HTTP.  Possible enum values:  - &#x60;\&quot;HTTP\&quot;&#x60; means that the scheme used will be http://  - &#x60;\&quot;HTTPS\&quot;&#x60; means that the scheme used will be https:// | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


