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
    p = sub, obj, acts

    [policy_effect]
    e = some(where (p.eft == allow))

    [matchers]
    m = keyMatch(r.sub, p.sub) && keyMatch(r.obj, p.obj) && (contains(p.acts, r.act) || p.acts == '*')
  policy.csv: |
    # The argo-server has read-only access
    p, serviceaccount:cluster-0:argo:argo-server, cluster-0:*, *
    # The SSO user "Cg0wLTM4NS0yODA4OS0wEgRtb2Nr" has read-only permissions.
    p, user:cluster-0:Cg0wLTM4NS0yODA4OS0wEgRtb2Nr, cluster-1:*:*, "get,list,watch"
kind: ConfigMap
metadata:
  name: argo-server-authz
```

Multi-Cluster Server authz differently depending on the configured auth-mode:

* `server` - the service account is `argo-server` so rules are configured using subject `serviceaccount:argo:argo-server`.
* `client` - no authz is provided, the supplied client token must be for the cluster specific in the request.
* `sso` - rules are configured using subject `user:${claims.subject}`.

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

Restart the controller:

```bash
kubectl rollout restart deploy/argo-server
```

```bash
# install resources into remote cluster
kubectl kustomize --load-restrictor=LoadRestrictionsNone manifests/quick-start/cluster-1 | kubectl --context=cluster-1 -n default apply -f -

# install profile into local cluster
argo cluster get-profile cluster-1 default argo-server.cluster-0 argo-server --server=https://`ipconfig getifaddr en0`:`kubectl config view --raw --minify --context=cluster-1|grep server|cut -c 29-` --insecure-skip-tls-verify | kubectl -n argo apply -f  -
```

To access the API, you should change your URLs: `/api/v1/{namespace}` becomes `/api/v2/{cluster/{namespace}`.
