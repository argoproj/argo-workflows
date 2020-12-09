# Disaster Recovery (DR)

We only store data in your Kubernetes cluster. You should consider backing this up regularly.

Exporting example:

```
kubectl get wf,cwf,cwft,wftmpl -o yaml > backup.yaml
```

Importing example:

```
kubectl apply -f backup.yaml

```

You should also back-up any SQL persistence you use regularly with whatever tool is provided with it.