

# IoArgoprojWorkflowV1alpha1PodGC

PodGC describes how to delete completed pods as they complete

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**deleteDelayDuration** | [**Duration**](Duration.md) |  |  [optional]
**labelSelector** | [**LabelSelector**](LabelSelector.md) |  |  [optional]
**strategy** | **String** | Strategy is the strategy to use. One of \&quot;OnPodCompletion\&quot;, \&quot;OnPodSuccess\&quot;, \&quot;OnWorkflowCompletion\&quot;, \&quot;OnWorkflowSuccess\&quot;. If unset, does not delete Pods |  [optional]



