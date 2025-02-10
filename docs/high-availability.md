# High-Availability (HA)

## Workflow Controller

In the event of a Workflow Controller pod failure, the replacement Controller pod will continue running Workflows when it is created.
In most cases, this short loss of Workflow Controller service may be acceptable.

If you run a single replica of the Workflow Controller, ensure that the [environment variable](environment-variables.md#controller) `LEADER_ELECTION_DISABLE` is set to `true` and that the pod uses the `workflow-controller` [Priority Class](https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/) included in the installation manifests.

By disabling the leader election process, you can avoid unnecessary communication with the Kubernetes API, which may become unresponsive when running Workflows at scale.

By using the `PriorityClass`, you can ensure that the Workflow Controller pod is scheduled before other pods in the cluster.

### Multiple Workflow Controller Replicas

It is possible to run multiple replicas of the Workflow Controller to provide high-availability
Ensure that leader election is enabled (either by omitting the `LEADER_ELECTION_DISABLE` or setting it to `false`).

Only one replica of the Workflow Controller will actively manage workflows at any given time.
The other replicas will be on standby, ready to take over if the active replica fails.
This means that you are guaranteeing resource allocations for replicas that are not actively contributing to the running of workflows.

The leader election process requires frequent communication with the Kubernetes API.
When running workflows at scale, the Kubernetes API may become unresponsive, causing the leader election to take longer than 10 seconds (`LEADER_ELECTION_RENEW_DEADLINE`) to respond, which will disrupt the controller.

Even with multiple replicas, a voluntary pod disruption can cause both replicas to be replaced simultaneously.
Use a [Pod Disruption Budget](https://kubernetes.io/docs/concepts/workloads/pods/disruptions/#pod-disruption-budgets) to prevent this.

### Considerations

A single replica of the Workflow Controller is recommended for most use cases due to:

- The time to re-provision the controller pod is often faster than the time for an existing pod to win a leader election, especially when the cluster is under load.
- You save on the cost of extra Kubernetes resource allocations that aren't being used.

## Argo Server

Run a minimum of two replicas, typically three, should be run, otherwise it may be possible that API and webhook requests are dropped.

!!! Tip
    Consider [spreading Pods across multiple availability zones](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/).
