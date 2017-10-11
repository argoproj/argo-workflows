# argo-lite

Argo-lite is a lightweight workflow engine that executes container-native workflows defined using [Argo YAML](https://argoproj.github.io/argo-site/docs/yaml/dsl_reference_intro.html).  Argo-lite implements the same APIs as [Argo](https://github.com/argoproj/argo) and is therefore compatible with the [Argo CLI](https://argoproj.github.io/argo-site/docs/dev-cli-reference.html) and Argo UI.  Argo-lite currently supports Docker and  Kubernetes as the backend container execution engines but should be easy to modify to work with nearly any container engine.  

## Argo-lite will be released in mid October.

Argo-lite is not yet fully tested and may crash under load. Early testing/contributions are very welcome.

## Why?

Argo-lite may be used to quicky experience [Argo](https://github.com/argoproj/argo) workflows without deploying a complete Kubernetes cluster or to debug Argo workflows locally on your laptop.

## Try it

### On your laptop:

1. Run argo-lite server:
```
docker run --rm -p 8080:8080  -v /var/run/docker.sock:/var/run/docker.sock -dt argoproj/argo-lite node /app/dist/main.js -u /app/dist/ui
```
2. Configure [Argo CLI](https://argoproj.github.io/docs/dev-cli-reference.html) to talk to your Argo-lite instance:

```
argo login --config argo-lite http://localhost:8080 --username test --password test
```

### On your kubernetes cluster:

1. Create Argo-lite deployment manually:

```
curl -o /tmp/argo.yaml https://raw.githubusercontent.com/argoproj/argo/master/lite/argo-lite.yaml && kubectl create -f /tmp/argo.yaml
```
or using helm:

```
helm repo add argo https://argoproj.github.io/argo-helm
helm install argo/argo-lite
```

2. Configure [Argo CLI](https://argoproj.github.io/docs/dev-cli-reference.html) to talk to your Argo-lite instance:

```
argo login --config argo-lite-kube <deployed argo-lite service URL> --username test --password test
```

### Sample sample workflows

Build argo-lite using argo-lite :-) ```git clone https://github.com/argoproj/argo.git && cd argo && argo job submit 'Argo Lite CI' --config argo-lite --local```

![alt text](./demo.gif "Logo Title Text 1")
