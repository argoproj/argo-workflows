

# Condition

Condition contains details for one aspect of the current state of this API Resource.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**lastTransitionTime** | **java.time.Instant** |  | 
**message** | **String** | message is a human readable message indicating details about the transition. This may be an empty string. | 
**observedGeneration** | **Integer** | observedGeneration represents the .metadata.generation that the condition was set based upon. For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date with respect to the current state of the instance. |  [optional]
**reason** | **String** | reason contains a programmatic identifier indicating the reason for the condition&#39;s last transition. Producers of specific condition types may define expected values and meanings for this field, and whether the values are considered a guaranteed API. The value should be a CamelCase string. This field may not be empty. | 
**status** | **String** | status of the condition, one of True, False, Unknown. | 
**type** | **String** | type of condition in CamelCase or in foo.example.com/CamelCase. | 



