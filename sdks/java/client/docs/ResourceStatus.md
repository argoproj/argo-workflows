

# ResourceStatus

ResourceStatus represents the status of a single resource allocated to a Pod.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **String** | Name of the resource. Must be unique within the pod and in case of non-DRA resource, match one of the resources from the pod spec. For DRA resources, the value must be \&quot;claim:&lt;claim_name&gt;/&lt;request&gt;\&quot;. When this status is reported about a container, the \&quot;claim_name\&quot; and \&quot;request\&quot; must match one of the claims of this container. | 
**resources** | [**List&lt;ResourceHealth&gt;**](ResourceHealth.md) | List of unique resources health. Each element in the list contains an unique resource ID and its health. At a minimum, for the lifetime of a Pod, resource ID must uniquely identify the resource allocated to the Pod on the Node. If other Pod on the same Node reports the status with the same resource ID, it must be the same resource they share. See ResourceID type definition for a specific format it has in various use cases. |  [optional]



