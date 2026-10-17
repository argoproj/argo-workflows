

# WorkloadReference

WorkloadReference identifies the Workload object and PodGroup membership that a Pod belongs to. The scheduler uses this information to apply workload-aware scheduling semantics.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **String** | Name defines the name of the Workload object this Pod belongs to. Workload must be in the same namespace as the Pod. If it doesn&#39;t match any existing Workload, the Pod will remain unschedulable until a Workload object is created and observed by the kube-scheduler. It must be a DNS subdomain. | 
**podGroup** | **String** | PodGroup is the name of the PodGroup within the Workload that this Pod belongs to. If it doesn&#39;t match any existing PodGroup within the Workload, the Pod will remain unschedulable until the Workload object is recreated and observed by the kube-scheduler. It must be a DNS label. | 
**podGroupReplicaKey** | **String** | PodGroupReplicaKey specifies the replica key of the PodGroup to which this Pod belongs. It is used to distinguish pods belonging to different replicas of the same pod group. The pod group policy is applied separately to each replica. When set, it must be a DNS label. |  [optional]



