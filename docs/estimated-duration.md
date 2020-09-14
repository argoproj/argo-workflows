# Estimated Duration

![alpha](assets/alpha.svg)

> v2.13 and after

If you run a workflow, the controller will calculate estimated duration by looking at the last time the workflow was run.

To determine which was the last run, we find the most recently completed workflow which has is also labelled with the same workflow template, cluster workflow template or cron workflow.
 
We query the [workflow archive](workflow-archive.md) first (if enabled) then Kubernetes if not, and this include both successful and failed workflows. 

If you've used tools like Jenkins, you'll know that that estimates can be inaccurate:

* The last time it ran, it failed early.
* A pod spent a long amount of time waiting to be scheduled.
* The workflow is non-deterministic, e.g. it uses `when` to execute different paths. 
* The workflow can vary is scale, e.g. sometimes it uses `withItems` and so sometimes run  100 nodes, sometimes a 1000.
* If the pod runtimes are unpredictable.
* The workflow template is parameterized, and this affect its duration.
  
