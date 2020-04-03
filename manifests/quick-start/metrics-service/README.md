This is copied verbatim as follows:

```
git clone -b https://github.com/kubernetes-sigs/metrics-server.git
metrics-server
git checkout v0.3.6
cp -R deploy/1.8+ ~/go/src/github.com/argoproj/argo/manifests/quick-start/base/metrics-service
```