

# IoArgoprojWorkflowV1alpha1Link

Link is a link to another app.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **String** | The name of the link, E.g. \&quot;Workflow Logs\&quot; or \&quot;Pod Logs\&quot; | 
**scope** | **String** | \&quot;workflow\&quot;, \&quot;pod\&quot;, \&quot;pod-logs\&quot;, \&quot;event-source-logs\&quot;, \&quot;sensor-logs\&quot;, \&quot;workflow-list\&quot; or \&quot;chat\&quot; | 
**target** | **String** | Target attribute specifies where a linked document will be opened when a user clicks on a link. E.g. \&quot;_blank\&quot;, \&quot;_self\&quot;. If the target is _blank, it will open in a new tab. | 
**url** | **String** | The URL. Can contain \&quot;${metadata.namespace}\&quot;, \&quot;${metadata.name}\&quot;, \&quot;${status.startedAt}\&quot;, \&quot;${status.finishedAt}\&quot; or any other element in workflow yaml, e.g. \&quot;${io.argoproj.workflow.v1alpha1.metadata.annotations.userDefinedKey}\&quot; | 



