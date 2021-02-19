# Release Instructions

Allow 1h to do a release.

## Preparation

* [ ] Cherry-pick your changes from master onto the release branch.
* [ ] The release branch should be green in CI before you start.

## Release

Releasing requires a clean tree state, so back-up any untracked files in your Git directory.

**Only once your files are backed up**, run:

    git clean -fdx  # WARNING: Will delete untracked files!

To generate new manifests and perform basic checks:

    make prepare-release -B VERSION=v2.11.5

Publish the images and local Git changes (disabling K3D as this is faster and more reliable for releases):

    make publish-release K3D=false VERSION=v2.11.5
    
Wait 1h to 2h.

* [ ] Check the images were pushed successfully.
* [ ] Check the correct versions are printed.
* [ ] Check the executor was correctly built.

```
docker run argoproj/argoexec:v2.11.5 version
docker run argoproj/workflow-controller:v2.11.5 version
docker run argoproj/argocli:v2.11.5 version
```

* [ ] Check the manifests contain the correct tags: https://raw.githubusercontent.com/argoproj/argo-workflows/v2.11.5/manifests/install.yaml

* [ ] Check the manifests apply: `kubectl -n argo apply -f https://raw.githubusercontent.com/argoproj/argo-workflows/v2.11.5/manifests/install.yaml`

### Release Notes

Create [the release](https://github.com/argoproj/argo-workflows/releases) in Github. You can get some text for this using [Github Toolkit](https://github.com/alexec/github-toolkit):

    ght relnote v2.7.3..v2.11.4

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

* [ ] Check the manifests contain the correct tags: https://raw.githubusercontent.com/argoproj/argo-workflows/stable/manifests/install.yaml

### Update Homebrew

If this is GA:

* [ ] Update the Homebrew formula.

```bash
export HOMEBREW_GITHUB_API_TOKEN=$GITHUB_TOKEN
brew bump-formula-pr argo --version 2.11.5
```

* [ ] Check that Homebrew was successfully updated after the PR was merged:
 
 ```
 brew upgrade argo
 /usr/local/bin/argo version
 ```

### Update Java SDK

If this is GA:

* [ ] Update the Java SDK formula.

```
git clone git@github.com:argoproj-labs/argo-client-java.git
cd argo-client-java
make publish VERSION=v2.11.5
```

* [ ] Check package published: https://github.com/argoproj-labs/argo-client-java/packages
