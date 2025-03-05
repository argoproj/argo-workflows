# High-Availability (HA)

By default, the Workflow Controller Pod(s) and the Argo Server Pod(s) do not have resource requests or limits configured.
Set resource requests to guarantee a resource allocation appropriate for your workloads.

When you use multiple replicas of the same deployment, spread the Pods across multiple availability zones.
At a minimum, ensure that the Pods are not scheduled on the same node.

Use a [Pod Disruption Budget](https://kubernetes.io/docs/concepts/workloads/pods/disruptions/#pod-disruption-budgets) to prevent all replicas from being replaced simultaneously.

## Workflow Controller

In the event of a Workflow Controller Pod failure, the replacement Controller Pod will continue running Workflows when it is created.
In most cases, this short loss of Workflow Controller service may be acceptable.

If you run a single replica of the Workflow Controller, ensure that the [environment variable](environment-variables.md#controller) `LEADER_ELECTION_DISABLE` is set to `true` and that the Pod uses the `workflow-controller` [Priority Class](https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/) included in the installation manifests.

By disabling the leader election process, you can avoid unnecessary communication with the Kubernetes API, which may become unresponsive when running Workflows at scale.

By using the `PriorityClass`, you can ensure that the Workflow Controller Pod is scheduled before other Pods in the cluster.

### Multiple Workflow Controller Replicas

It is possible to run multiple replicas of the Workflow Controller to provide high-availability.
Ensure that leader election is enabled (either by omitting the `LEADER_ELECTION_DISABLE` or setting it to `false`).

Only one replica of the Workflow Controller will actively manage Workflows at any given time.
The other replicas will be on standby, ready to take over if the active replica fails.
This means that you are guaranteeing resource allocations for replicas that are not actively contributing to the running of Workflows.

The leader election process requires frequent communication with the Kubernetes API.
When running Workflows at scale, the Kubernetes API may become unresponsive, causing the leader election to take longer than 10 seconds (`LEADER_ELECTION_RENEW_DEADLINE`) to respond, which will disrupt the controller.

### Considerations

A single replica of the Workflow Controller is recommended for most use cases due to:

- The time taken to re-provision the controller Pod often being faster than the time for an existing Pod to win a leader election, especially when the cluster is under load.
- Saving on the cost of extra Kubernetes resource allocations that aren't being used.

## Argo Server

Run a minimum of two replicas, typically three, to avoid dropping API and webhook requests.
