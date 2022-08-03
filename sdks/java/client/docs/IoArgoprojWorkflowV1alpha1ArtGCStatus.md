

# IoArgoprojWorkflowV1alpha1ArtGCStatus

map ArtifactGC Pod name to phase

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**notSpecified** | **Boolean** | if this is true, we already checked to see if we need to do it and we don&#39;t |  [optional]
**podsRecouped** | **Map&lt;String, Boolean&gt;** | have completed Pods been processed? (mapped by Pod name) |  [optional]
**strategiesProcessed** | **Map&lt;String, Boolean&gt;** | have Pods been started to perform this strategy? |  [optional]



