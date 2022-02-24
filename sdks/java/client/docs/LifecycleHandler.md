

# LifecycleHandler

LifecycleHandler defines a specific action that should be taken in a lifecycle hook. One and only one of the fields, except TCPSocket must be specified.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**exec** | [**ExecAction**](ExecAction.md) |  |  [optional]
**httpGet** | [**HTTPGetAction**](HTTPGetAction.md) |  |  [optional]
**tcpSocket** | [**TCPSocketAction**](TCPSocketAction.md) |  |  [optional]



