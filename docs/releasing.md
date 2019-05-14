# Release Instructions

1. Update CHANGELOG.md with changes in the release

2. Update VERSION with new tag

3. Update codegen, manifests with new tag

```bash
make codegen manifests IMAGE_NAMESPACE=argoproj IMAGE_TAG=vX.Y.Z
```

4. Commit VERSION and manifest changes

```bash
git add .
git commit -m "Update version to vX.Y.Z"
```

5. git tag the release

```bash
git tag vX.Y.Z
```

6. Build both the controller and UI release

In argo repo:
```bash
make release IMAGE_NAMESPACE=argoproj IMAGE_TAG=vX.Y.Z
```

In argo-ui repo:
```bash
IMAGE_NAMESPACE=argoproj IMAGE_TAG=vX.Y.Z yarn docker
```

8. If successful, publish the release:
```bash
export ARGO_RELEASE=vX.Y.Z
docker push argoproj/workflow-controller:${ARGO_RELEASE}
docker push argoproj/argoexec:${ARGO_RELEASE}
docker push argoproj/argocli:${ARGO_RELEASE}
docker push argoproj/argoui:${ARGO_RELEASE}
```

9. Push commits and tags to git. Run the following in both the argo and argo-ui repos:

In argo repo:
```bash
git push upstream
git push upstream ${ARGO_RELEASE}
```

In argo-ui repo:
```bash
git push upstream ${ARGO_RELEASE}
```

10. Draft GitHub release with the content from CHANGELOG.md, and CLI binaries produced in the `dist` directory

* https://github.com/argoproj/argo/releases/new
