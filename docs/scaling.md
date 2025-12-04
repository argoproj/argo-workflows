# Scaling

For running large workflows, you'll typically need to scale the controller to match.

## Horizontally Scaling

You cannot horizontally scale the controller.
The controller supports having a hot-standby for [High Availability](high-availability.md#workflow-controller).

## Vertically Scaling

You can scale the controller vertically in these ways:

### Container Resource Requests

If you observe the Controller using its total CPU or memory requests, you should increase those.

### Adding Goroutines to Increase Concurrency

If you have sufficient CPU cores, you can take advantage of them with more goroutines:

- If you have many Workflows and you notice they're not being reconciled fast enough, increase `--workflow-workers`.
- If you're using `TTLStrategy` in your Workflows and you notice they're not being deleted fast enough, increase `--workflow-ttl-workers`.
- If you're using `PodGC` in your Workflows and you notice the Pods aren't being deleted fast enough, increase `--pod-cleanup-workers`.

> v3.5 and after

- If you're using a lot of `CronWorkflows` and they don't seem to be firing on time, increase `--cron-workflow-workers`.

### K8S API Client Side Rate Limiting

The Kubernetes client library used by the Workflow Controller rate limits the number of API requests that can be sent to the Kubernetes API server.
This rate limiting helps prevent overwhelming the API server with too many requests at once.

If you frequently see messages similar to this in the Controller log (issued by the library):

```txt
Waited for 7.090296384s due to client-side throttling, not priority and fairness, request: GET:https://10.100.0.1:443/apis/argoproj.io/v1alpha1/namespaces/argo/workflowtemplates/s2t
```

Or, in >= v3.5, if you see warnings similar to this (could be any CR, not just `WorkflowTemplate`):

```txt
Waited for 7.090296384s, request:GET:https://10.100.0.1:443/apis/argoproj.io/v1alpha1/namespaces/argo/workflowtemplates/s2t
```

These messages indicate that the Controller is being throttled by the client-side rate limiting.

#### Adjusting Rate Limiting

By using cluster-wide observability tooling, you can determine whether or not your Kubernetes API server can handle more requests.
You can increase the rate limits by adjusting the `--qps` and `--burst` arguments for the Controller:

- `--qps`: This argument sets the average number of queries per second allowed by the Kubernetes client.
The default value is 20.
- `--burst`: This argument sets the number of queries per second the client can send before it starts enforcing the qps limit.
The default value is 30.
Typically, burst should be greater than qps.

By increasing these values, you can allow the Controller to send more requests to the API server, reducing the likelihood of throttling.

##### Example Configuration

To increase the rate limits, you might set the arguments as follows:

```yaml
args:
  - --qps=50
  - --burst=75
```

This configuration allows the Controller to send an average of 50 queries per second, with a burst capacity of 75 queries per second before throttling is enforced.

It is important to note that increasing these values can increase the load on the Kubernetes API server and that you must observe your Kubernetes API under load in order to determine whether or not the values you have chosen are correct for your needs.
It is not possible to provide a one-size-fits-all recommendation for these values.

### Rate Limiting Pod Creation

The Workflow Controller is responsible for the requesting the creation of Pods from the Kubernetes API.
Creating pods is an expensive/heavy operation, and requesting too many Pods at once can in turn cause the Kubernetes API server to become overwhelmed.
To mitigate this, you can set rate limit how many Pods the Workflow Controller requests by adjusting the `limit` and `burst` values in the [Workflow Controller ConfigMap](workflow-controller-configmap.yaml).

```yaml
  # Globally limits the rate at which pods are created.
  # This is intended to mitigate flooding of the Kubernetes API server by workflows with a large amount of
  # parallel nodes.
  resourceRateLimit: |
    limit: 10
    burst: 25
```

- `limit`: This sets the average number of Pod creation requests per second..
- `burst`: This sets the number of Pods per second the Controller can create before it starts enforcing the `limit`.
Typically, burst should be greater than the limit.

By using cluster-wide observability tooling, you can determine whether or not your Kubernetes API server can handle more Pod creation requests.
It is important to note that increasing these values can increase the load on the Kubernetes API server and that you must observe your Kubernetes API under load in order to determine whether or not the values you have chosen are correct for your needs.
It is not possible to provide a one-size-fits-all recommendation for these values.

!!! Note
    Despite the name, this rate limit only applies to the creation of Pods and not the creation of other Kubernetes resources (for example, ConfigMaps or PersistentVolumeClaims).

### Offloading Workflow Task Results to a Secondary Kubernetes API Server

Workflow Task Results are how Argo Workflows tracks outputs of pods and passes them between tasks (in a DAG) and steps.
They are provided as a Custom Resource Definition (CRD) within the Argo Workflows installation, as `WorkflowTaskResults`, with the Argo executor creating and updating them, and the Workflow Controller reading them.
It is possible that with many workflows the Kubernetes API will be overwhelmed due to the creation and deletion of many `WorkflowTaskResults` on the cluster.
To solve this, the Workflow Controller ConfigMap can specify an `OffloadTaskResultsConfig`.

#### POC Setup (Not for upstream docs)

The goal is to have a fully functional Kubernetes API endpoint that stores Argo's `WorkflowTaskResults` in its own data store.
Conceptually, we will be running a lightweight sub-cluster within the main cluster, in a similar way to tools like `vcluster`.
For this, we will run a Kubernetes API Server (Service and Deployment) and point it to an `etcd` Service/Deployment for its backend storage.
This is loosely how the Kubernetes Control Plane itself runs -- for more information, take a look at the [Kubernetes Components](https://kubernetes.io/docs/concepts/overview/components/) documentation.

##### Running a Kubernetes API Server

###### Run an `etcd` instance

We deploy a single-node `etcd`. The API server uses it exactly like the real Kubernetes control plane would.

| Flag                                                   | Why we need it                                          |
| ------------------------------------------------------ | ------------------------------------------------------- |
| `--data-dir=/var/lib/etcd`                             | Local storage. We use `emptyDir:` for ephemeral POC.    |
| `--advertise-client-urls` / `--listen-client-urls`     | Expose the client API on port 2379.                     |
| `--listen-peer-urls` / `--initial-advertise-peer-urls` | Required even for a single-member “cluster”.            |
| `--initial-cluster`                                    | Defines the cluster membership. Required syntactically. |

The Service simply exposes port 2379 inside the namespace so the API server can reach it at `http://argo-wtr-etcd.argo.svc:2379`.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: argo-wtr-etcd
  namespace: argo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: argo-wtr-etcd
  template:
    metadata:
      labels:
        app: argo-wtr-etcd
    spec:
      containers:
      - name: etcd
        image: quay.io/coreos/etcd:v3.6.6
        command:
        - etcd
        - --name=argo-wtr-etcd
        - --data-dir=/var/lib/etcd
        - --advertise-client-urls=http://0.0.0.0:2379
        - --listen-client-urls=http://0.0.0.0:2379
        - --listen-peer-urls=http://0.0.0.0:2380
        - --initial-advertise-peer-urls=http://0.0.0.0:2380
        - --initial-cluster=argo-wtr-etcd=http://0.0.0.0:2380
        ports:
        - containerPort: 2379
        - containerPort: 2380
        volumeMounts:
        - name: data
          mountPath: /var/lib/etcd
      volumes:
      - name: data
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: argo-wtr-etcd
  namespace: argo
spec:
  selector:
    app: argo-wtr-etcd
  ports:
  - port: 2379
    targetPort: 2379
    name: client
```

###### Set Up Certs & API Server Security

A full kube-apiserver normally requires multiple certificates, CA bundles, front-proxy certs, and authentication plugins.

For this POC we run with the absolute minimum we can get away with:

| File                 | Purpose                                                                                    |
| -------------------- | ------------------------------------------------------------------------------------------ |
| `tls.crt`, `tls.key` | Server certificate & private key for HTTPS endpoint (`--secure-port=6443`).                 |
| `serviceaccount.key` | Used both as the *public* and *private* key for signing service account tokens.            |
| `tokens.csv`         | Static token authentication. Used so kubectl can authenticate without bootstrap machinery. |

We create `tls.crt` and `tls.key` using:

```console
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout tls.key -out tls.crt -subj "/CN=argo-wtr-apiserver"
```

We create `serviceaccount.key` using:

```console
openssl genrsa -out serviceaccount.key 2048
```

`tokens.csv` contains a static token authentication, where the file format is `<token>,<user>,<uid>,<group1>,<group2>,...`.

Copy these values to the ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: certs-and-keys
  namespace: argo
data:
  serviceaccount.key: |
    -----BEGIN PRIVATE KEY-----
    <snipped>
    -----END PRIVATE KEY-----
  tls.crt: |
    -----BEGIN CERTIFICATE-----
    <snipped>
    -----END CERTIFICATE-----

  tls.key: |
    -----BEGIN PRIVATE KEY-----
    <snipped>
    -----END PRIVATE KEY-----
  tokens.csv: |
    mytoken,admin,1,"system:masters"
```

###### Run the kube-apiserver

| Flag                                                | Why                                                 |
| --------------------------------------------------- | --------------------------------------------------- |
| `--etcd-servers=http://argo-wtr-etcd.argo.svc:2379` | Backend database.                                   |
| `--secure-port=6443`                                 | Only expose HTTPS; insecure port removed in >=1.31. |
| `--tls-cert-file`, `--tls-private-key-file`         | Required since insecure-port is gone.               |
| `--token-auth-file=/var/run/kubernetes/tokens.csv`  | Simplest auth flow for kubectl.                     |
| `--service-account-key-file`                        | Needed even if we don’t actually use SA tokens.     |
| `--service-account-signing-key-file`                | Required in 1.20+ to serve the SA issuer.           |
| `--service-account-issuer`                          | Must match what your workloads use when validating. |
| `--authorization-mode=AlwaysAllow`                  | Disables RBAC entirely.                             |
| `--enable-admission-plugins=NamespaceLifecycle`     | Default admission plugin required for namespace-scoped CRDs and is on by default in upstream. |

###### Apply the `WorkflowTaskResults` CRD

<!-- Doesn't seem to be needed? `kit` tasks forwards everything for you? -->
Once the API server is running, we can port forward it and apply the CRD directly to it.

Port forward in a separate terminal:

```console
kubectl -n argo port-forward service/argo-wtr-apiserver 6443:6443;
```

And then run the `apply`:

```console
kubectl \
  --server=https://localhost:6443 \
  --token=mytoken \
  --insecure-skip-tls-verify=true \
  apply -f manifests/base/crds/minimal/argoproj.io_workflowtaskresults.yaml
```

Also create the `argo` namespace in the API server:

```console
kubectl \
  --server=https://localhost:6443 \
  --token=mytoken \
  --insecure-skip-tls-verify=true \
  create ns argo
```


###### Optional convenience: use a Config for `kubectl` and `k9s`

To save writing out the args to `kubectl` and `k9s`, you can use this Config:

```yaml
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://localhost:6443
    insecure-skip-tls-verify: true
  name: argo-wtr-cluster
users:
- name: argo-wtr-user
  user:
    token: mytoken
contexts:
- context:
    cluster: argo-wtr-cluster
    user: argo-wtr-user
  name: argo-wtr-context
current-context: argo-wtr-context
```

And run commands like:

```console
KUBECONFIG=api-server-kubeconfig.yaml kubectl get ns
KUBECONFIG=api-server-kubeconfig.yaml ./k9s
```

(Download `k9s` to the container if using Dev Containers.)

##### Set Up the Controller Config

The final step is to tell our Workflows Controller about the offloadTaskResults config.
Based on the above config with the server at `https://localhost:6443`, we can use this `ConfigMap` as `manifests/base/workflow-controller/workflow-controller-configmap.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:
  offloadTaskResults: |
    enabled: true
    APIServer: https://localhost:6443
```

And finally (with the api-server still port-forwarded) run `make start` to run the workflow controller with workflowtaskresult offloading!

## Sharding

### One Install Per Namespace

Rather than running a single installation in your cluster, run one per namespace using the `--namespaced` flag.

### Instance ID

Within a cluster you can use instance ID to run many Argo instances within a cluster.
You can run each instance in a separate, or the same namespace.

For each instance, edit the [`workflow-controller-configmap.yaml`](workflow-controller-configmap.yaml) to set an `instanceID`.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:
    instanceID: i1
```

#### CLI Usage

When you are using the CLI, set the `--instanceid` to interact with a specific instance.
For example, to submit a Workflow defined in `my-wf.yaml`:

```bash
argo --instanceid i1 submit my-wf.yaml
```

#### Declarative Usage

You can also declare which instance to use in the each of the Custom Resources. For example, on a `Workflow`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
    labels:
        workflows.argoproj.io/controller-instanceid: i1
```

It can also be useful, when defining a `WorkflowTemplate` to use `templateDefaults`

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
    labels:
        workflows.argoproj.io/controller-instanceid: i1
    name: example
spec:
    templateDefaults:
        metadata:
            labels:
                workflows.argoproj.io/controller-instanceid: i1
```

### Maximum Recursion Depth

In order to protect users against infinite recursion, the controller has a default maximum recursion depth of 100 calls to templates.

This protection can be disabled with the [environment variable](environment-variables.md#controller) `DISABLE_MAX_RECURSION=true`

### Caching Semaphore Limit ConfigMap Requests

By default the controller will reload the ConfigMap(s) referenced by a semaphore from kube every time that workflow is queued. If you notice high latency from queuing workflows leveraging semaphores you can cache semaphore limits by editing the `semaphoreLimitCacheSeconds` parameter in [`workflow-controller-configmap.yaml`](workflow-controller-configmap.yaml).

Note that this will mean that Argo will not immediately pick up changes to your config map limits.

## Miscellaneous

See also [Running At Massive Scale](running-at-massive-scale.md).
