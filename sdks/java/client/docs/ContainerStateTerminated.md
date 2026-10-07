

# ContainerStateTerminated

ContainerStateTerminated is a terminated state of a container.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**containerID** | **String** | Container&#39;s ID in the format &#39;&lt;type&gt;://&lt;container_id&gt;&#39; |  [optional]
**exitCode** | **Integer** | Exit status from the last termination of the container | 
**finishedAt** | **java.time.Instant** |  |  [optional]
**message** | **String** | Message regarding the last termination of the container |  [optional]
**reason** | **String** | (brief) reason from the last termination of the container |  [optional]
**signal** | **Integer** | Signal from the last termination of the container |  [optional]
**startedAt** | **java.time.Instant** |  |  [optional]



