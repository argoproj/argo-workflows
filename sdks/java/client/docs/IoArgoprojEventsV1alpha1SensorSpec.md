

# IoArgoprojEventsV1alpha1SensorSpec


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**circuit** | **String** | Circuit is a boolean expression of dependency groups Deprecated: will be removed in v1.5, use Switch in triggers instead. |  [optional]
**dependencies** | [**List&lt;IoArgoprojEventsV1alpha1EventDependency&gt;**](IoArgoprojEventsV1alpha1EventDependency.md) | Dependencies is a list of the events that this sensor is dependent on. |  [optional]
**dependencyGroups** | [**List&lt;IoArgoprojEventsV1alpha1DependencyGroup&gt;**](IoArgoprojEventsV1alpha1DependencyGroup.md) | DependencyGroups is a list of the groups of events. |  [optional]
**errorOnFailedRound** | **Boolean** | ErrorOnFailedRound if set to true, marks sensor state as &#x60;error&#x60; if the previous trigger round fails. Once sensor state is set to &#x60;error&#x60;, no further triggers will be processed. |  [optional]
**eventBusName** | **String** |  |  [optional]
**replicas** | **Integer** |  |  [optional]
**template** | [**IoArgoprojEventsV1alpha1Template**](IoArgoprojEventsV1alpha1Template.md) |  |  [optional]
**triggers** | [**List&lt;IoArgoprojEventsV1alpha1Trigger&gt;**](IoArgoprojEventsV1alpha1Trigger.md) | Triggers is a list of the things that this sensor evokes. These are the outputs from this sensor. |  [optional]



