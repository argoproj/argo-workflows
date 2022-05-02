# Artifact Visualization

> since v3.4

Use cases:

* Comparing ML pipeline runs from generated charts
* Visualizing end results of ML pipeline runs
* Debugging workflows where visual artifacts are the most helpful

[![Demo](https://img.youtube.com/vi/whoRfYY9Fhk/0.jpg)](https://youtu.be/whoRfYY9Fhk)

* Artifacts now appear as elements in the workflow DAG that you can click on.
* When you click on the artifact, a panel appears.
* This first time this opens, it shows explanatory text that helps you understand if you might need to change their
  workflows to use this new feature.
* Known file types such as images, text or HTML are displayed in an inline frame (`iframe`).
* Artifacts are sandboxed using a Content-Security-Policy that prevents JavaScript execution.
* JSON, being popular, is displayed in an special viewer.

To start, you should take a look at
a [fully formed example](https://github.com/argoproj/argo-workflows/blob/master/examples/artifacts-workflowtemplate.yaml)
.

## Compressed Artifacts

By default artifacts are compressed as a `.tgz`. Viewing of `.tgz` is not supported in the user interface. Only files
that were stored uncompressed are supported. Set `archive` to `none` to prevent compression.

```yaml
- name: html
  # ...
  archive:
    none: { }
```

## File Type

File type is determine by the file extension of artifact's key. Not from the artifact name and not from the path. Make
sure the key has the correct extension:

```yaml
- name: html
  s3:
    key: index.html
```

## HTML

You can create reports using HTML artifacts, which include charts and graphs produced by your workflow.

## Security

### Malicious Artifacts

A **malicious artifact** is a HTML artifact that attempts to use JavaScript to perform UI actions, such as creating or
deleting workflows.

We assume that artifacts are not trusted, so by default, artifacts are served with a `Content-Security-Policy` that
disables JavaScript.

This is similar to what happens when you include third-party scripts, such as analytic tracking, in your website.
However, those tracking codes are normally served from a different domain to your main website. Artifacts are server
from the same origin, so normal browser controls are not secure enough.

### Sub-Path Access

Previously, users can access the artifacts of any workflows they can access. To allow HTML files to link to other files
within their tree, you can now access any sub-paths of the artifact.

Example:

The artifact produces a folder is an S3 bucket named `my-bucket`, with a key `my-key`. You can also access anything
matching `my-key/*` too.
