Description: Add a MiniStack component for the quick-start artifact repository
Authors: [Nahuel Nucera](https://github.com/Nahuel990)
Component: Build and Development
Issues: 17030

The quick-start runs MinIO for the S3 artifact repository, pinned to an image published in November 2022.
MinIO has since stopped publishing community edition images and its repository is archived, so there is no upstream path for a future CVE.

A new component at `manifests/components/ministack` serves the artifact repository with MiniStack instead.
It replaces the MinIO Deployment and Service, repoints the artifact repository endpoints at `ministack:4566`, and creates the `my-bucket` bucket on container start.
The existing `my-minio-cred` secret is reused unchanged.

The component is opt-in and the MinIO path is untouched, so existing quick-start users are unaffected.
To use it, build the new variant with `kubectl kustomize manifests/quick-start/ministack`, or add `../../components/ministack` to the components list of your own overlay.
