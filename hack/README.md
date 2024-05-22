# `hack/` scripts

This directory contains various automation scripts or "hacks".

The directory structure roughly follows the top-level structure:

- [`hack/api/`](api/) has scripts for [`api/`](../api/)
- [`hack/docs/`](docs/) has scripts for [`docs/`](../docs/)
- [`hack/git/`](git/) has scripts for `.git/`
- [`hack/manifests/`](manifests/) has scripts for [`manifests/`](../manifests/)

## Modifications

When adding or deleting files in this directory, make sure to also update [the `Makefile`](../Makefile) and the [GitHub Actions](../.github/workflows/)
