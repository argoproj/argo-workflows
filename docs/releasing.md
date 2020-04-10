# Release Instructions

Allow 1h to do a release.

## Preparation

Cherry-pick your changes from master onto the release branch.

The release branch should be green [in CircleCI](https://app.circleci.com/github/argoproj/argo/pipelines) before you start.

## Release

To generate new manifests and perform basic checks:

    make prepare-release VERSION=v2.7.2

Publish the images and local Git changes:

    make publish-release

Create [the release](https://github.com/argoproj/argo/releases) in Github. You can get some text for this using [Github Toolkit](https://github.com/alexec/github-toolkit):

    ght relnote v2.7.1..v2.7.2

Release notes checklist:

* [ ] All breaking changes are listed with migration steps
* [ ] The release notes identify every publicly known vulnerability with a CVE assignment 

If this is GA:

* [ ] Update the `stable` tag
* [ ] Update the [Homebrew tap](https://github.com/argoproj/homebrew-tap).
 
