# Release Instructions

1. Update CHANGELOG.md with changes in the release

2. Update VERSION with new tag

```bash
export VERSION=vX.Y.Z
```

3. Update codegen, manifests with new tag

```bash
make codegen manifests IMAGE_TAG=$VERSION
```

4. Commit VERSION and manifest changes

```bash
git add .
git commit -m "Update version to $VERSION"
```

5. Tag the release

```bash
git tag vX.Y.Z
```

6. Build both the release

In argo repo:

```bash
make release IMAGE_TAG=$VERSION
```

8. If successful, publish the release:

```bash
docker push argoproj/workflow-controller:${VERSION}
docker push argoproj/argoexec:${VERSION}
docker push argoproj/argocli:${VERSION}
docker push argoproj/argo-server:${VERSION}
```

9. Push commits and tags to git. Run the following in the argo repos:

In Argo repo:

```bash
git push upstream
git push upstream ${VERSION}
git tag stable
git push upstream stable
```

10. Draft GitHub release with the content from CHANGELOG.md, and CLI binaries produced in the `dist` directory

* https://github.com/argoproj/argo/releases/new
