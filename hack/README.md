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

The helper makes at most three attempts, waiting 5 seconds and then 10 seconds between retryable failures.
It retries recognized HTTP 408, 429, 500, 502, 503 and 504 diagnostics, HTTP/2 stream `INTERNAL_ERROR` responses from the peer, and temporary connection or name-resolution errors.
Other errors, including authentication failures, missing packages, checksum failures and signal-style exit statuses, fail without retries.
Recognized permanent diagnostics take precedence when the same attempt also reports a transient error.
Each attempt is classified using only its own output, which remains visible while the command runs.
The final command exit status is preserved and temporary diagnostic files are removed.
Existing package caches, integrity checks and CI step timeouts remain in effect.
Do not wrap builds, tests or non-idempotent commands with this helper.

Run `go test ./hack/downloads -count=1` to test the helper, including recovery from a local HTTP 502 using a real Yarn install when Yarn is available.

## Modifications

When adding or deleting files in this directory, make sure to also update [the `Makefile`](../Makefile) and the [GitHub Actions](../.github/workflows/)
