

# IoArgoprojEventsV1alpha1EventDependencyFilter

EventDependencyFilter defines filters and constraints for a io.argoproj.workflow.v1alpha1.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**context** | [**IoArgoprojEventsV1alpha1EventContext**](IoArgoprojEventsV1alpha1EventContext.md) |  |  [optional]
**data** | [**List&lt;IoArgoprojEventsV1alpha1DataFilter&gt;**](IoArgoprojEventsV1alpha1DataFilter.md) |  |  [optional]
**exprs** | [**List&lt;IoArgoprojEventsV1alpha1ExprFilter&gt;**](IoArgoprojEventsV1alpha1ExprFilter.md) | Exprs contains the list of expressions evaluated against the event payload. |  [optional]
**time** | [**IoArgoprojEventsV1alpha1TimeFilter**](IoArgoprojEventsV1alpha1TimeFilter.md) |  |  [optional]



