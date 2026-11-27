# `hack/` scripts

This directory contains various automation scripts or "hacks".

The directory structure roughly follows the top-level structure:

- [`hack/api/`](api/) has scripts for [`api/`](../api/)
- [`hack/docs/`](docs/) has scripts for [`docs/`](../docs/)
- [`hack/git/`](git/) has scripts for `.git/`
- [`hack/manifests/`](manifests/) has scripts for [`manifests/`](../manifests/)

## Retrying package downloads

Use `sh hack/retry-download.sh COMMAND [ARG ...]` for idempotent package-download commands that can fail because of a temporary registry or network error.
The Go module downloads and Yarn installs in CI image builds, the main CI workflow, the docs workflow and the UI Make targets use this helper.
For example:

```sh
sh hack/retry-download.sh go mod download
sh hack/retry-download.sh yarn --cwd ui install
```

The helper retries any failed command up to three attempts total, waiting 5 seconds and then 10 seconds between attempts.
It does not classify errors: permanent failures are retried too, avoiding dependence on package-manager diagnostic strings.
Command output is passed through directly, and the final command exit status is preserved unless the retry delay fails, in which case the helper returns `sleep`'s status.
Existing package caches, integrity checks and CI step timeouts remain in effect.
Do not wrap builds, tests or non-idempotent commands with this helper.

## Modifications

When adding or deleting files in this directory, make sure to also update [the `Makefile`](../Makefile) and the [GitHub Actions](../.github/workflows/)
