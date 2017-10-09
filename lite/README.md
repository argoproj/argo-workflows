# argo-lite

Argo-lite is a lightweight workflow engine that executes container-native workflows defined using [Argo YAML](https://argoproj.github.io/argo-site/docs/yaml/dsl_reference_intro.html).  Argo-lite implements the same APIs as [Argo](https://github.com/argoproj/argo) and is therefore compatible with the [Argo CLI](https://argoproj.github.io/argo-site/docs/dev-cli-reference.html) and Argo UI.  Argo-lite currently supports Docker and  Kubernetes as the backend container execution engines but should be easy to modify to work with nearly any container engine.  

## Argo-lite will be released in mid October.

Argo-lite is not yet fully tested and may crash under load. Early testing/contributions are very welcome.

Several features of the full Argo workflow engine are not yet supported by Argo-lite.

- [x] ~~Kubernetes integration~~
- [x] ~~API to access artifacts~~
- [x] ~~[Dynamic fixtures](https://argoproj.github.io/argo-site/docs/yaml/fixture_template.html)~~
- [ ] Add Argo UI into Argo-lite distribution
- [x] [Docker-in-Docker](https://argoproj.github.io/argo-site/docs/yaml/argo_tutorial_2_create_docker_image_build_workflow.html)
- [ ] No unit or e2e tests

## Why?

Argo-lite may be used to quicky experience [Argo](https://github.com/argoproj/argo) workflows without deploying a complete Kubernetes cluster or to debug Argo workflows locally on your laptop.

## Deploy it:

Three ways of running  Argo-lite.

1. Run everything using locally installed Docker

```
docker run --rm -p 8080:8080  -v /var/run/docker.sock:/var/run/docker.sock -dt argoproj/argo-lite node /app/dist/main.js
```

2. Run argo-lite locally but use an existing kubernetes cluster as the backend container engine

```
docker run --rm -p 8080:8080 -v <path-to-your-kube-config>:/cluster.conf -it argoproj/argo-lite node /app/dist/main.js --engine kubernetes --config /cluster.conf
```

3. Run everything on an existing kubernetes cluster

```
git clone git@github.com:argoproj/argo-lite.git && cd argo-lite && kubectl create -f argo-lite.yaml
```

## Try it:

Install and configure the [Argo CLI](https://argoproj.github.io/argo-site/docs/dev-cli-reference.html).

Build argo-lite using argo-lite :-)

```
cd argo-lite/.argo
argo job submit checkout-build --config mini --local
```

![alt text](./demo.gif "Logo Title Text 1")
