

# V1Lifecycle

Lifecycle describes actions that the management system should take in response to container lifecycle events. For the PostStart and PreStop lifecycle handlers, management of the container blocks until the action is complete, unless the container process fails, in which case the handler is aborted.
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**postStart** | [**V1Handler**](V1Handler.md) |  |  [optional]
**preStop** | [**V1Handler**](V1Handler.md) |  |  [optional]



