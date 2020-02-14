# V1Lifecycle

Lifecycle describes actions that the management system should take in response to container lifecycle events. For the PostStart and PreStop lifecycle handlers, management of the container blocks until the action is complete, unless the container process fails, in which case the handler is aborted.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**post_start** | [**V1Handler**](V1Handler.md) |  | [optional] 
**pre_stop** | [**V1Handler**](V1Handler.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


