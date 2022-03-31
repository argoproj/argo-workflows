# Multi-Cluster Server

Argo v3.4 introduces the ability to run a single control-plane API and UI server for all your clusters.

Update the config map with permissions:

```yaml
apiVersion: v1
data:
  model.conf: |-
    [request_definition]
    r = sub, obj, act

    [policy_definition]
    p = sub, obj, act

    [role_definition]
    g = _, _

    [policy_effect]
    e = some(where (p.eft == allow))

    [matchers]
    m = g(r.sub, p.sub) && keyMatch(r.sub, p.sub) && keyMatch(r.obj, p.obj) && (contains(p.act, r.act) || p.act == '*')
  policy.csv: |
    # The argo-server has read-write access in cluster-0.
    p, serviceaccount:cluster-0:argo:argo-server, cluster-0:*, *

    # Users have read-write access in cluster-0.
    p, user:cluster-0:*, cluster-0:*:*, *
kind: ConfigMap
metadata:
  name: argo-server-authz
```

The argo server must be configured with the name of its own cluster:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: argo-server
spec:
  template:
    spec:
      containers:
        - name: argo-server
          env:
            - name: ARGO_CLUSTER
              value: cluster-0
```

Resources must be deployed into the remote cluster, and the profile imported into the primary cluster, for example:

```bash
# install resources into remote cluster
kubectl kustomize --load-restrictor=LoadRestrictionsNone manifests/quick-start/cluster-1 | kubectl --context=cluster-1 -n default apply -f -

# install profile into primary cluster
argo cluster get-profile cluster-1 default argo-server.cluster-0 argo-server --server=https://`ipconfig getifaddr en0`:`kubectl config view --raw --minify --context=cluster-1|grep server|cut -c 29-` --insecure-skip-tls-verify | kubectl -n argo apply -f  -
```

Restart the server:

```bash
kubectl rollout restart deploy/argo-server
```

To access the API, you should change your URLs: `/api/v1/workflows/argo`
becomes `/api/v1/workflows/argo?cluster=cluster-1`.

## Authorization

We use Casbin for access control. The default model.conf should work for most cases. You will need to write your own
policy.csv to replace the default one.

```csv
# The argo-server has read-write access in cluster-0.
p, serviceaccount:cluster-0:argo:argo-server, cluster-0:*, *

# Users have read-write access in cluster-0.
p, user:cluster-0:*, cluster-0:*, *
```

For example:

```csv
# The argo-server has read-write access in cluster-0.
p, serviceaccount:cluster-0:argo:argo-server, cluster-0:*, *

# The argo-server service account has read-only permissions in cluster-1.
p, serviceaccount:cluster-0:argo:argo-server, cluster-1:*:*, "get,list,watch"

# The SSO user 'Cg0wLTM4NS0yODA4OS0wEgRtb2Nr' has read-only permissions.
p, user:cluster-0:Cg0wLTM4NS0yODA4OS0wEgRtb2Nr, cluster-1:*:*, "get,list,watch"

# The SSO group 'authors' has read-only permissions.
p, group:cluster-0:authors, cluster-0:*:*, "get,list,watch"
```

Each entry is a subject,object,action.

Subject:

* `serviceaccount:{cluster}:{namespace}:{name}` - for `client` and `server` auth-modes.
* `user:{cluster}:{name}` - for SSO.
* `group:{cluster}:{name}` - for SSO when the `groups` scope is enabled.

Object:

* `{cluster}:{namespace}:{resource}` - the object of the request. See table below.

Action:

* A single value, e.g. `list`.
* A comma-separated list, e.g. `get,list,watch`.
* `*` for anything.

| resource                    | act                                                                                       |
|-----------------------------|-------------------------------------------------------------------------------------------|
| archivedworkflowlabelkeys   | list                                                                                      |
| archivedworkflowlabelvalues | list                                                                                      |
| archivedworkflows           | delete,get,list,resubmit,retry                                                            |
| clusterworkflowtemplates    | create,delete,get,lint,list,update                                                        |
| cronworkflows               | create,delete,get,lint,list,resume,suspend,update                                         |
| events                      | receive,watch                                                                             |
| eventsources                | create,delete,get,list,update,watch                                                       |
| eventsourceslogs            | watch                                                                                     |
| infos                       | get                                                                                       |
| inputartifactbyuids         | get                                                                                       |
| inputartifacts              | get                                                                                       |
| outputartifactbyuids        | get                                                                                       |
| outputartifacts             | get                                                                                       |
| pipelinelogs                | watch                                                                                     |
| pipelines                   | delete,get,list,restart,watch                                                             |
| podlogs                     | watch                                                                                     |
| sensors                     | create,delete,get,list,update,watch                                                       |
| sensorslogs                 | watch                                                                                     |
| steps                       | watch                                                                                     |
| userinfos                   | get                                                                                       |
| versions                    | get                                                                                       |
| workfloweventbindings       | list                                                                                      |
| workflowlogs                | watch                                                                                     |
| workflows                   | create,delete,get,lint,list,resubmit,resume,retry,set,stop,submit,suspend,terminate,watch |
| workflowtemplates           | create,delete,get,lint,list,update                                                        |

## Scaling

An argo-server serving data from multiple clusters will need to be scale-up to match load.

## Reliability

The argo-server is stateless. When serving data from clusters other than the primary cluster, requests will be disrupted
by network problems more often.
