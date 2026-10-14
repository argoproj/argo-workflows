

# IoArgoprojWorkflowV1alpha1RetryNodeAntiAffinity

RetryNodeAntiAffinity avoids running steps on the hosts that previous attempts ran on. In order to identify hosts, it uses \"kubernetes.io/hostname\".

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**type** | **String** | Type determines whether a retry excludes previously used hosts outright (\&quot;Required\&quot;, the default) or merely de-prioritises them (\&quot;Preferred\&quot;). Use \&quot;Preferred\&quot; when a retry should still be able to run on a previously used host once every eligible host has been tried, rather than staying pending. |  [optional]



