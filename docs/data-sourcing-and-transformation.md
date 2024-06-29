# Data Sourcing and Transformations

> v3.1 and after

We have intentionally made this feature available with only bare-bones functionality. Our hope is that we are able to build this feature with our community's feedback. If you have ideas and use cases for this feature, please open an [enhancement proposal](https://github.com/argoproj/argo-workflows/issues/new?assignees=&labels=enhancement&template=enhancement_proposal.md) on GitHub.

Additionally, please take a look at our current ideas at the bottom of this document.

## Introduction

Users often source and transform data as part of their workflows. The `data` template provides first-class support for these common operations.

`data` templates can best be understood by looking at a common data sourcing and transformation operation in `bash`:

```bash
find -r . | grep ".pdf" | sed "s/foo/foo.ready/"
```

Such operations consist of two main parts:

* A "source" of data: `find -r .`
* A series of "transformations" which transform the output of the source serially: `| grep ".pdf" | sed "s/foo/foo.ready/"`

This operation, for example, could be useful in sourcing a potential list of files to be processed and filtering and manipulating the list as desired.

In Argo, this operation would be written as:

```yaml
- name: generate-artifacts
  data:
    source:             # Define a source for the data, only a single "source" is permitted
      artifactPaths:    # A predefined source: Generate a list of all artifact paths in a given repository
        s3:             # Source from an S3 bucket
          bucket: test
          endpoint: minio:9000
          insecure: true
          accessKeySecret:
            name: my-minio-cred
            key: accesskey
          secretKeySecret:
            name: my-minio-cred
            key: secretkey
    transformation:     # The source is then passed to be transformed by transformations defined here
      - expression: "filter(data, {# endsWith \".pdf\"})"
      - expression: "map(data, {# + \".ready\"})"
```

## Spec

A `data` template must always contain a `source`. Current available sources:

* `artifactPaths`: generates a list of artifact paths from the artifact repository specified

A `data` template may contain any number of transformations (or zero). The transformations will be applied serially in order. Current available transformations:

* `expression`: an [expression](variables.md#expression). The data is accessible in the `data` variable (see example above).

    We understand that the `expression` transformation is limited. We intend to greatly expand the functionality of this template with our community's feedback. Please see the link at the top of this document to submit ideas or use cases for this feature.
