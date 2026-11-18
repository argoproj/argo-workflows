

# PodExtendedResourceClaimStatus

PodExtendedResourceClaimStatus is stored in the PodStatus for the extended resource requests backed by DRA. It stores the generated name for the corresponding special ResourceClaim created by the scheduler.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**requestMappings** | [**List&lt;ContainerExtendedResourceRequest&gt;**](ContainerExtendedResourceRequest.md) | RequestMappings identifies the mapping of &lt;container, extended resource backed by DRA&gt; to  device request in the generated ResourceClaim. | 
**resourceClaimName** | **String** | ResourceClaimName is the name of the ResourceClaim that was generated for the Pod in the namespace of the Pod. | 



