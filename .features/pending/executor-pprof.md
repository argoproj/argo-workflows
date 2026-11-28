Description: Profiling endpoints on the workflow executor
Authors: [Abhishek Sharma](https://github.com/abhisheksharma2411)
Component: Telemetry
Issues: 17283

The `wait` and `supervisor` containers now expose the Go `pprof` endpoints when you set `ARGO_PPROF` to `true`, as the controller and the server already do.
Use this when an executor container grows in memory or CPU over the life of a step, which you cannot diagnose from outside the process.
The endpoints listen on loopback inside the pod, so they are not reachable from the rest of the cluster, and `kubectl port-forward` still reaches them.
Set `ARGO_PPROF_PORT` if one of your own containers listens on the default port `6060`.
