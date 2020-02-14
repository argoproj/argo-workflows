# V1OwnerReference

OwnerReference contains enough information to let you identify an owning object. An owning object must be in the same namespace as the dependent, or be cluster-scoped, so there is no namespace field.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_version** | **str** | API version of the referent. | [optional] 
**block_owner_deletion** | **bool** |  | [optional] 
**controller** | **bool** |  | [optional] 
**kind** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**uid** | **str** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


