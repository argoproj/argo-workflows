Description: Automatic pod restarts also cover pods that kubelet rejects for resources or topology
Authors: [Simon Domke](https://github.com/usimd)
Component: General
Issues: 17476

With `failedPodRestart` enabled, the controller now also restarts pods that kubelet rejects at admission with `OutOf<resource>`, for example `OutOfcpu`, `OutOfmemory` or `OutOfnvidia.com/gpu`, or with `TopologyAffinityError`.
These pods never started, so the controller recreates them the same way as for `Evicted` or `UnexpectedAdmissionError`, up to `maxRestarts` times.
You no longer need a `retryStrategy` to recover from these rejections.
