# High-Availability (HA)

## Workflow Controller

Before v3.0, only one controller could run at once. (If it crashed, Kubernetes would start another pod.)

> v3.0

For many users, a short loss of workflow service may be acceptable - the new controller will just continue running
workflows if it restarts.  However, with high service guarantees, new pods may take too long to start running workflows.
You should run two replicas, and one of which will be kept on hot-standby.

A voluntary pod disruption can cause both replicas to be replaced at the same time. You should use a Pod Disruption
Budget to prevent this and Pod Priority to recover faster from an involuntary pod disruption:

* [Pod Disruption Budget](https://kubernetes.io/docs/concepts/workloads/pods/disruptions/#pod-disruption-budgets)
* [Pod Priority](https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/)

## Argo Server

> v2.6

Run a minimum of two replicas, typically three, should be run, otherwise it may be possible that API and webhook requests are dropped.

!!! Tip
    Consider using [multi AZ-deployment using pod anti-affinity](https://www.verygoodsecurity.com/blog/posts/kubernetes-multi-az-deployments-using-pod-anti-affinity).
