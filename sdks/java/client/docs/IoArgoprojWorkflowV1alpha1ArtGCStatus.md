

# IoArgoprojWorkflowV1alpha1ArtGCStatus

ArtGCStatus maintains state related to ArtifactGC

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**notSpecified** | **Boolean** | if this is true, we already checked to see if we need to do it and we don&#39;t |  [optional]
**podsRecouped** | **Map&lt;String, Boolean&gt;** | have completed Pods been processed? (mapped by Pod name) used to prevent re-processing the Status of a Pod more than once |  [optional]
**strategiesProcessed** | **Map&lt;String, Boolean&gt;** | have Pods been started to perform this strategy? (enables us not to re-process what we&#39;ve already done) |  [optional]



