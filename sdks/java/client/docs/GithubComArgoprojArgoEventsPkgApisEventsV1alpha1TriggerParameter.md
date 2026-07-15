

# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**dest** | **String** | Dest is the JSONPath of a resource key. A path is a series of keys separated by a dot. The colon character can be escaped with &#39;.&#39; The -1 key can be used to append a value to an existing array. See https://github.com/tidwall/sjson#path-syntax for more information about how this is used. |  [optional]
**operation** | **String** | Operation is what to do with the existing value at Dest, whether to &#39;prepend&#39;, &#39;overwrite&#39;, or &#39;append&#39; it. |  [optional]
**src** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource.md) |  |  [optional]



