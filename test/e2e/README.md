# E2E Testing

1. Run `make start port-forward`
2. Either (a) run your test in your IDE or (b) run `make test`.

Notes:

* Everything runs in the `argo` namespace (including MinIO). 
* For speed, please only use `docker/whalesay:latest`. 
* Test can take longer on CI. Adds 5s to timeout values.

## Debugging E2E Tests

### Logs

```
make logs
```

### Accessing Argo UI

```
kubectl -n argo port-forward deployment/argo-ui 8001:8001
```

Then open http://localhost:8001

### Accessing MinIO

```
kubectl -n argo port-forward pod/minio 9000:9000
```

Then open http://localhost:9000 using admin/password.

### Expose Database

#### Postgres

```
kubectl -n argo port-forward $(kubectl -n argo get pod -l app=postgres -o name) 5432:5432
```

Add to `/etc/hosts`:

```
127.0.0.1 postgres
```

#### MySQL

```
kubectl -n argo port-forward $(kubectl -n argo get pod -l app=mysql -o name) 3306:3306
```

Add to `/etc/hosts`:

```
127.0.0.1 mysql
```

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
kubectl -n argo scale deploy/argo-server --replicas 0
```

Kill any port forwards on 2746.

The run `cmd/server/main.go` using these arguments, which enable debug logging, and make sure you use locally build image:

```
--loglevel debug --enable-client-auth
```


### To Update The Executor

If you're making changes to the executor, run:

```
make executor-image DEV_IMAGE=true IMAGE_PREFIX=argoproj/ IMAGE_TAG=dev 
```

### To Switch Between Postgres and MySQL

Edit `test/e2e/manifest/workflow-controller-config.yaml` and comment/un-comment correct section.

```
kubectl -n argo apply test/e2e/manifest/workflow-controller-config.yaml
```

Then either for Postgres: 

```
kubectl -n argo scale deploy/mysql --replicas 0
kubectl -n argo scale deploy/postgres --replicas 1
```

Or for MySQL

```
kubectl -n argo scale deploy/postgres --replicas 0
kubectl -n argo scale deploy/mysql --replicas 1
```

To access the Postgres database as follows:

```
kubectl exec -ti $(kubectl get pod -l app=postgres -o name) -- psql -U postgres
select * from argo_workflows;
```

To access the MySQL database as follows:

```
kubectl exec -ti $(kubectl get pod -l app=mysql -o name) -- mysql -u mysql -ppassword argo
select * from argo_workflows;
```

