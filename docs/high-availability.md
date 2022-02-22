# High-Availability (HA)

## Workflow Controller

Only one controller can run at once. If it crashes, Kubernetes will start another pod.

> v3.0 

For many users, a short loss of workflow service maybe acceptable - the new controller will just continue running workflows if it restarts.  However, with high service guarantees, new pods may take too long to start running workflows. You should run two replicas, and one of which will be kept on hot-standby.

## Argo Server

> v2.6

Run a minimum of two replicas, typically three, should be run, otherwise it maybe possible that API and webhook requests are dropped.

!!! Tip
    Consider using [multi AZ-deployment using pod anti-affinity](https://www.verygoodsecurity.com/blog/posts/kubernetes-multi-az-deployments-using-pod-anti-affinity). 