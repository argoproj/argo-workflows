# Release Instructions

1. Update CHANGELOG.md with changes in the release

2. Update VERSION with new tag

3. Update codegen, manifests with new tag

```
make codegen manifests IMAGE_NAMESPACE=argoproj IMAGE_TAG=vX.Y.Z
```

4. Commit VERSION and manifest changes

```
git add .
git commit -m "Update version to vX.Y.Z"
```

5. git tag the release

```
git tag vX.Y.Z
```

6. Build the release

```
make release IMAGE_NAMESPACE=argoproj IMAGE_TAG=vX.Y.Z
```

7. If successful, publish the release:
```
export ARGO_RELEASE=vX.Y.Z
docker push argoproj/workflow-controller:${ARGO_RELEASE}
docker push argoproj/argoexec:${ARGO_RELEASE}
docker push argoproj/argocli:${ARGO_RELEASE}
git push upstream ${ARGO_RELEASE}
```

8. Draft GitHub release with the content from CHANGELOG.md, and CLI binaries produced in the `dist` directory

* https://github.com/argoproj/argo/releases/new
