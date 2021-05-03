# Release Instructions

## Release

### 1. Cherry-pick Issue

Create a cherry-pick issue to allow the team and community to comment on the release contents.

1. Locate the previous cherry-pick issue
2. Get the hash of the most recent commit still available on the previous issue
3. Generate new issue contents:
    
    ```sh
    $ git checkout master # Ensure we are on master
    $ git log --pretty=format:"%an: %s %h"  [COMMIT_HASH]..HEAD
    ```
4. Create a new issue on GitHub with the title `[VERSION] cherry-pick` (e.g. `v3.0.2 cherry-pick`) and the generated commits
as content.

### 2. Cherry-pick to Release Branch

Once the team and community is satisfied with the commits to be cherry-picked, cherry-pick them into the appropriate
release branch. There should be a single release branch per minor release (e.g. `release-3.0`, `release-3.1`, etc.)

1. Checkout the release branch and cherry-pick commits

    ```sh
    $ git checkout relesae-3.0
    $ git cherry-pick [COMMIT_IDS...]
    ```

2. Hope for few merge conflicts!

3. Once done cherry-picking, push the release branch to ensure the branch can build and all tests pass.

### 3. Prepare the Release

#### NOTE: Releasing for `v2`

`v2` releases still depend on the previous repository name (`github.com/argoproj/argo`). To release for `v2`,
make a local clone of the repository under the name `argo`:

```shell
$ pwd
/Users/<user>/go/src/github.com/argoproj/argo-workflows
$ cd ..
$ cp -r argo-workflows argo
$ cd argo
```

Then follow all the normal steps. You should delete the `argo` folder once the release is done to avoid confusion and conflicts.

#### Preparing the release

1. Releasing requires a clean tree state, so back-up any untracked files in your Git directory.
   
   **Only once your files are backed up**, run:
      ```shell
       $ git clean -fdx  # WARNING: Will delete untracked files!
      ```

2. To generate new manifests and perform basic checks:
   
      ```shell
      $ make prepare-release -B VERSION=v3.0.3
      ```

3. Once done, push the release branch and ensure the branch is green and all tests pass.

      ```shell
      $ git push
      ```

4. Publish the images and local Git changes (disabling K3D as this is faster and more reliable for releases):

      ```shell
      $ make publish-release K3D=false VERSION=v3.0.3
      ```

5. Wait 1h to 2h.

### 4. Ensure the Release Succeeded

1. Check the images were pushed successfully. Ensure the `GitTreeState` is `Clean`.
   ```sh
   $ docker run argoproj/argoexec:v3.0.3 version
   $ docker run argoproj/workflow-controller:v3.0.3 version
   $ docker run argoproj/argocli:v3.0.3 version
   ```
   
1. Check the correct versions are printed. Ensure the `GitTreeState` is `Clean`.
   ```sh
   $ ./dist/argo-darwin-amd64 version
   ```

1. Check the manifests contain the correct tags (search for `v3.0.3`): https://raw.githubusercontent.com/argoproj/argo-workflows/v3.0.3/manifests/install.yaml

1. Check the manifests apply: `kubectl -n argo apply -f https://raw.githubusercontent.com/argoproj/argo-workflows/v3.0.3/manifests/install.yaml`

### 5. Release Notes

Create [the release](https://github.com/argoproj/argo-workflows/releases) on Github. You can get some text for this using [Github Toolkit](https://github.com/alexec/github-toolkit):

    ght relnote v3.0.2..v3.0.3

Alternatively, you can get it manually with the following commands

```shell
$ git checkout release-3.0 # Ensure we are on the release branch

# Change names (v3.0.2 is the PREVIOUS release in this example)
$ git log --pretty=format:"- %s"  v3.0.2..v3.0.3 | pbcopy  

# Contributor names (v3.0.2 is the PREVIOUS release in this example)
$ git log --pretty=format:"- %an"  v3.0.2..v3.0.3 | sort | uniq
```

The release title should be the version number (e.g. `v3.0.3`) and nothing else.

Release notes checklist:

* All breaking changes are listed with migration steps
* The release notes identify every publicly known vulnerability with a CVE assignment

### 6. Upload Binaries and SHA256 Sums To Github

After running `make publish-relesae`, you will have the zipped binaries and SHA256 sums in your local.

Open them with:

```shell
$ open dist
```

Upload only the zipped binaries (`.gz` suffix) and SHA256 sums (`.sha256` suffix) to GitHub. There should be 12 uploaded files in total.

### 6. Update Stable Tag

If this is GA:

Update the `stable` tag

```
git tag -f stable
git push -f origin stable
```

Check the manifests contain the correct tags: https://raw.githubusercontent.com/argoproj/argo-workflows/stable/manifests/install.yaml

### 7. Update Homebrew

If this is GA:

Update the Homebrew formula.

```bash
export HOMEBREW_GITHUB_API_TOKEN=$GITHUB_TOKEN
brew bump-formula-pr argo --version 2.11.5
```

Check that Homebrew was successfully updated after the PR was merged:
 
 ```
 brew upgrade argo
 /usr/local/bin/argo version
 ```

### 8. Update Java SDK

If this is GA:

Update the Java SDK formula.

```
git clone git@github.com:argoproj-labs/argo-client-java.git
cd argo-client-java
make publish VERSION=v2.11.5
```

Check package published: https://github.com/argoproj-labs/argo-client-java/packages

### 9. Publish Release

Finally, press publish on the GitHub release. Congrats, you're done!