

# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SensorSpec


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**dependencies** | [**List&lt;GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventDependency&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventDependency.md) | Dependencies is a list of the events that this sensor is dependent on. |  [optional]
**errorOnFailedRound** | **Boolean** | ErrorOnFailedRound if set to true, marks sensor state as &#x60;error&#x60; if the previous trigger round fails. Once sensor state is set to &#x60;error&#x60;, no further triggers will be processed. |  [optional]
**eventBusName** | **String** |  |  [optional]
**loggingFields** | **Map&lt;String, String&gt;** |  |  [optional]
**replicas** | **Integer** |  |  [optional]
**revisionHistoryLimit** | **Integer** |  |  [optional]
**template** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Template**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Template.md) |  |  [optional]
**triggers** | [**List&lt;GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Trigger&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Trigger.md) | Triggers is a list of the things that this sensor evokes. These are the outputs from this sensor. |  [optional]



