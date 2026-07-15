

# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1PayloadField

PayloadField binds a value at path within the event payload against a name.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **String** | Name acts as key that holds the value at the path. |  [optional]
**path** | **String** | Path is the JSONPath of the event&#39;s (JSON decoded) data key Path is a series of keys separated by a dot. A key may contain wildcard characters &#39;*&#39; and &#39;?&#39;. To access an array value use the index as the key. The dot and wildcard characters can be escaped with &#39;\\\\&#39;. See https://github.com/tidwall/gjson#path-syntax for more information on how to use this. |  [optional]



