# Stress Testing

Create a cluster in [`jesse-sb` project](https://console.cloud.google.com/access/iam?cloudshell=false&project=jesse-sb)
with at least 21 nodes.

Install `gcloud` binary.

Login to GCP: `gloud auth login`

Get your KUBECONFIG, something like:

```
gcloud container clusters get-credentials cluster-1 --zone us-central1-c --project jesse-sb
```

Run `make start PROFILE=stress`.

Make sure pods are running:

```
kubectl get deployments
```

If this fails, just try running it again.

* Open http://localhost:2746/workflows and check it loads and that you can run a workflow.
* Open http://localhost:9090/metrics and check you can see the Prometheus metrics.
* Open http://localhost:9091/graph and check you can see a Prometheus graph.
* Open http://localhost:6060/debug/pprof and check you can access pprof.

Run `go run ./test/stress/tool -n 1`  to test with a small number workflows and make sure they complete.

Do you get `ImagePullBackOff`? Make sure image is `argoproj/argosay:v2`
in  `kubectl -n argo edit workflowtemplate massive-workflow`.

You can
use [this Tab Auto Refresh Chrome extension](https://chrome.google.com/webstore/detail/tab-auto-refresh/oomoeacogjkolheacgdkkkhbjipaomkn)
to auto-refresh the page.

Run `go run ./test/stress/tool -n 10000` to run a large number of workflows.

* Use Prometheus to see how many Kubernetes API requests are being made.
* Use PProf to see if there any any hot spots:

```
go tool pprof -png http://localhost:6060/debug/pprof/allocs
go tool pprof -png http://localhost:6060/debug/pprof/heap
go tool pprof -png http://localhost:6060/debug/pprof/profile
```

