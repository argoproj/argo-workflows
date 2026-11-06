

# IoArgoprojWorkflowV1alpha1PooledTaskStatus

PooledTaskStatus is the controller's record of a pooled node's task.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**activeDeadlineSeconds** | **Integer** | ActiveDeadlineSeconds limits how long the task may run once claimed, as dispatched. |  [optional]
**claimUID** | **String** | ClaimUID is the UID of the WorkflowTaskResult the claiming worker created. |  [optional]
**conditions** | [**List&lt;Condition&gt;**](Condition.md) | Conditions record the task&#39;s hand-off to a worker: Claimed and Interrupted. Its state is derived from them (State). Change them only through Enqueue, Claim, Release and Interrupt. |  [optional]
**deadline** | **java.time.Instant** |  |  [optional]
**enqueuedAt** | **java.time.Instant** |  | 
**poolHash** | **String** | PoolHash is the hash of the pool the task was dispatched to. | 
**poolName** | **String** | PoolName is the name of the pool the task was dispatched to. | 
**releases** | **Integer** | Releases is how many times a worker has released the task. |  [optional]
**taskName** | **String** | TaskName is the task&#39;s &#x60;node.taskName&#x60;. | 
**taskSetUID** | **String** | TaskSetUID is the UID of the task&#39;s WorkflowTaskSet. Only results that carry it are applied to the node. | 
**workerPodName** | **String** | WorkerPodName is the name of the claiming worker pod. |  [optional]
**workerPodUID** | **String** | WorkerPodUID is the UID of the claiming worker pod. |  [optional]
**workerRestartCount** | **Integer** | WorkerRestartCount is the claiming pod&#39;s worker container restart count when the claim was first seen. |  [optional]



