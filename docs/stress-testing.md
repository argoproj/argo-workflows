# Stress Testing

Install `gcloud` binary.

```bash
# Login to GCP:
gcloud auth login

# Set-up your config (if needed):
gcloud config set project alex-sb

# Create a cluster (default region is us-west-2, if you're not in west of the USA, you might want at different region):
gcloud container clusters create-auto argo-workflows-stress-1

# Get credentials:
gcloud container clusters get-credentials argo-workflows-stress-1                             

# Install workflows (If this fails, try running it again):
make start PROFILE=stress

# Make sure pods are running:
kubectl get deployments

# Run a test workflow:
argo submit examples/hello-world.yaml --watch
```

Checks

* Open <http://localhost:2746/workflows> and check it loads and that you can run a workflow.
* Open <http://localhost:9090/metrics> and check you can see the Prometheus metrics.
* Open <http://localhost:9091/graph> and check you can see a Prometheus graph. You can
  use [this Tab Auto Refresh Chrome extension](https://chrome.google.com/webstore/detail/tab-auto-refresh/oomoeacogjkolheacgdkkkhbjipaomkn)
  to auto-refresh the page.
* Open <http://localhost:6060/debug/pprof> and check you can access `pprof`.

Run `go run ./test/stress/tool -n 10000` to run a large number of workflows.

Check Prometheus:

1. See how many Kubernetes API requests are being made. You will see about one `Update workflows`
   per reconciliation, multiple `Create pods`. You should expect to see one `Get workflowtemplates` per workflow (done
   on first reconciliation). Otherwise, if you see anything else, that might be a problem.
2. How many errors were logged? `log_messages{level="error"}` What was the cause?

Check PProf to see if there any any hot spots:

```bash
go tool pprof -png http://localhost:6060/debug/pprof/allocs
go tool pprof -png http://localhost:6060/debug/pprof/heap
go tool pprof -png http://localhost:6060/debug/pprof/profile
```

## Clean-up

```bash
gcloud container clusters delete argo-workflows-stress-1
```
