

# IoArgoprojWorkflowV1alpha1TTLStrategy

TTLStrategy is the strategy for the time to live depending on if the workflow succeeded or failed

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**secondsAfterCompletion** | **Integer** | SecondsAfterCompletion is the number of seconds to live after completion |  [optional]
**secondsAfterError** | **Integer** | SecondsAfterError is the number of seconds to live after error |  [optional]
**secondsAfterFailure** | **Integer** | SecondsAfterFailure is the number of seconds to live after failure |  [optional]
**secondsAfterSuccess** | **Integer** | SecondsAfterSuccess is the number of seconds to live after success |  [optional]



