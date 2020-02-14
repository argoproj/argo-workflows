

# V1OwnerReference

OwnerReference contains enough information to let you identify an owning object. An owning object must be in the same namespace as the dependent, or be cluster-scoped, so there is no namespace field.
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**apiVersion** | **String** | API version of the referent. |  [optional]
**blockOwnerDeletion** | **Boolean** |  |  [optional]
**controller** | **Boolean** |  |  [optional]
**kind** | **String** |  |  [optional]
**name** | **String** |  |  [optional]
**uid** | **String** |  |  [optional]



