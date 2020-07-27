# Release Instructions

Allow 1h to do a release.

## Preparation

Cherry-pick your changes from master onto the release branch.

The release branch should be green in CI before you start.

## Release

To generate new manifests and perform basic checks:

    make prepare-release VERSION=v2.7.2

Publish the images and local Git changes (disabling K3D as this is faster and more reliable for releases):

    make publish-release K3D=false VERSION=v2.7.2
    
* [ ] Check the images were pushed successfully.

```
docker pull argoproj/workflow-controller:v2.7.2
docker pull argoproj/argoexec:v2.7.2
docker pull argoproj/argocli:v2.7.2
```

* [ ] Check the correct versions are printed:

```
docker run argoproj/workflow-controller:v2.7.2 version
docker run argoproj/argoexec:v2.7.2 version
docker run argoproj/argocli:v2.7.2 version
```

* [ ] Check the manifests contain the correct tags: https://raw.githubusercontent.com/argoproj/argo/v2.7.2/manifests/install.yaml

* [ ] Check the manifests apply: `kubectl -n argo apply -f https://raw.githubusercontent.com/argoproj/argo/v2.7.2/manifests/install.yaml`

### Release Notes

Create [the release](https://github.com/argoproj/argo/releases) in Github. You can get some text for this using [Github Toolkit](https://github.com/alexec/github-toolkit):

    ght relnote v2.7.1..v2.7.2

Release notes checklist:

* [ ] All breaking changes are listed with migration steps
* [ ] The release notes identify every publicly known vulnerability with a CVE assignment 

### Update Stable Tag

If this is GA:

* [ ] Update the `stable` tag

```
git tag -f stable
git push -f origin stable
```

* [ ] Check the manifests contain the correct tags: https://raw.githubusercontent.com/argoproj/argo/stable/manifests/install.yaml

### Update Homebrew

If this is GA:

* [ ] Update the Homebrew formula.

```bash
brew bump-formula-pr argo --version $VERSION
```

* [ ] Check that Homebrew was successfully updated after the PR was merged:
 
 ```
 brew upgrade argo
 argo version
 ```

