Description: Support for rootPath parameter to enable subpath deployments without requiring reverse proxy rewrites, ensuring
Author: [jiwlee97](https://github.com/jiwlee97)
Component: General
Issues: [#7767](https://github.com/argoproj/argo-workflows/issues/7767)

### Problem

When deploying Argo Workflows behind reverse proxies with path-based routing (e.g., AWS ALB), the current --base-href flag only updates UI asset paths and leaves API endpoints unchanged.
This causes API calls to fail with 404 errors unless the proxy supports path rewrites — which ALB does not.

### Solution

Introduce a new --root-path flag that:

- Prefixes all server endpoints (API, artifacts, OAuth, metrics, UI) with the given path
- Works independently or together with --base-href
- Removes the need for reverse proxy rewrites in subpath deployments

### Implementation

The --root-path flag provides a unified path prefix for both UI and API, aligning Argo Workflows with Argo CD’s subpath deployment behavior and maintaining full backward compatibility with existing --base-href configurations.
