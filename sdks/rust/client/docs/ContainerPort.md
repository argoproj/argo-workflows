# ContainerPort

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**container_port** | **i32** | Number of port to expose on the pod's IP address. This must be a valid port number, 0 < x < 65536. | 
**host_ip** | Option<**String**> | What host IP to bind the external port to. | [optional]
**host_port** | Option<**i32**> | Number of port to expose on the host. If specified, this must be a valid port number, 0 < x < 65536. If HostNetwork is specified, this must match ContainerPort. Most containers do not need this. | [optional]
**name** | Option<**String**> | If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services. | [optional]
**protocol** | Option<**String**> | Protocol for port. Must be UDP, TCP, or SCTP. Defaults to \"TCP\".  Possible enum values:  - `\"SCTP\"` is the SCTP protocol.  - `\"TCP\"` is the TCP protocol.  - `\"UDP\"` is the UDP protocol. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


