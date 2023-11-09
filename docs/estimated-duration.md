# Estimated Duration

> v2.12 and after

When you run a workflow, the controller will try to estimate its duration.

This is based on the most recently successful workflow submitted from the same workflow template, cluster workflow template or cron workflow.

To get this data, the controller queries the Kubernetes API first (as this is faster) and then [workflow archive](workflow-archive.md) (if enabled).

If you've used tools like Jenkins, you'll know that that estimates can be inaccurate:

* A pod spent a long amount of time pending scheduling.
* The workflow is non-deterministic, e.g. it uses `when` to execute different paths.
* The workflow can vary is scale, e.g. sometimes it uses `withItems` and so sometimes run  100 nodes, sometimes a 1000.
* If the pod runtimes are unpredictable.
* The workflow is parametrized, and different parameters affect its duration.
  