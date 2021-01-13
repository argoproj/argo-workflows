# Stress Testing

Create a cluster in [`jesse-sb` project](https://console.cloud.google.com/access/iam?cloudshell=false&project=jesse-sb).

Install `gcloud` binary.

Login to GCP: `gloud auth login`

Connect to your new cluster.

Make sure you've logged in to Docker Hub: `docker login`

Run `make start PROFILE=stress IMAGE_NAMESPACE=alexcollinsintuit DOCKER_PUSH=true`.

If this fails, just try running it again.

Open http://localhost:2746 and check you can run a workflow.

Open `test/stress/main.go` and run it with a small number (e.g. 10) workflows and make sure they complete.

Do you get `ImagePullBackOff`? Make sure image is `argoproj/argosay:v2` in  `kubectl -n argo edit workflowtemplate massive-workflow`.

Open http://localhost:9091/graph.

You can use [this Tab Auto Refresh Chrome extension](https://chrome.google.com/webstore/detail/tab-auto-refresh/oomoeacogjkolheacgdkkkhbjipaomkn) to auto-refresh the page.

Open `test/stress/main.go` and run it with a large number (e.g. 10000).

Use Prometheus to analyse this.

Finally, you can capture PProf using `./hack/capture-pprof.sh`.

