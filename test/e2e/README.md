# E2E Testing

1. Run `make start-e2e`.
2. Either (a) run your test in your IDE or (b) run `make test`.

Notes:

* Everything runs in the `argo` namespace (including MinIO). 
* For speed, please only use `docker/whalesay:latest`. 
* Test can take longer on CI. Adds 5s to timeout values.

## Debugging E2E Tests

### Accessing MinIO

Firstly enable port-forwarding:

```
kubectl -n argo port-forward pod/minio 9000:9000
```

Then open http://localhost:9000 using admin/password.

### Running Controller In Your IDE
 
If you want to run the controller in your IDE (e.g. to debug it), firstly scale down the controller:

```
kubectl -n argo scale deploy/workflow-controller --replicas 0
```

The run `cmd/workflow-controller/main.go` using these arguments, which enable debug logging, and make sure you use locally build image:

```
--loglevel debug --executor-image argoproj/argoexec:dev --executor-image-pull-policy Never
```

### Running The Argo Server In Your IDE

```
kubectl scale deploy/argo-server --replicas 0
```

The run `cmd/server/main.go` using these arguments, which enable debug logging, and make sure you use locally build image:

```
--loglevel debug --insecure
```


### To Update The Executor

If you're making changes to the executor, run:

```
make executor-image DEV_IMAGE=true IMAGE_PREFIX=argoproj/ IMAGE_TAG=dev 
```