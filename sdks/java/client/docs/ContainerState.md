

# ContainerState

ContainerState holds a possible state of container. Only one of its members may be specified. If none of them is specified, the default one is ContainerStateWaiting.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**running** | [**ContainerStateRunning**](ContainerStateRunning.md) |  |  [optional]
**terminated** | [**ContainerStateTerminated**](ContainerStateTerminated.md) |  |  [optional]
**waiting** | [**ContainerStateWaiting**](ContainerStateWaiting.md) |  |  [optional]



