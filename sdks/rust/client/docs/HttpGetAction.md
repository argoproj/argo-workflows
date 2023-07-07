# HttpGetAction

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**host** | Option<**String**> | Host name to connect to, defaults to the pod IP. You probably want to set \"Host\" in httpHeaders instead. | [optional]
**http_headers** | Option<[**Vec<crate::models::HttpHeader>**](HTTPHeader.md)> | Custom headers to set in the request. HTTP allows repeated headers. | [optional]
**path** | Option<**String**> | Path to access on the HTTP server. | [optional]
**port** | **String** |  | 
**scheme** | Option<**String**> | Scheme to use for connecting to the host. Defaults to HTTP.  Possible enum values:  - `\"HTTP\"` means that the scheme used will be http://  - `\"HTTPS\"` means that the scheme used will be https:// | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


