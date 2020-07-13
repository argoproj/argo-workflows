# Kubectl

You can also create Workflows directly with `kubectl`. However, the Argo CLI offers extra features
that `kubectl` does not, such as YAML validation, workflow visualization, parameter passing, retries
and resubmits, suspend and resume, and more.

```sh
kubectl create -n argo -f https://raw.githubusercontent.com/argoproj/argo/master/examples/hello-world.yaml
kubectl get wf -n argo
kubectl get wf hello-world-xxx -n argo
kubectl get po -n argo --selector=workflows.argoproj.io/workflow=hello-world-xxx
kubectl logs hello-world-yyy -c main -n argo
```

