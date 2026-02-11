Description: Add HTTP proxy support to Argo CLI
Authors: [Shimako55](https://github.com/shimako55)
Component: CLI
Issues: 10794

Add `--proxy-url` flag to Argo CLI commands to support HTTP proxy connections.
This allows users to connect to Argo Server or Kubernetes API through a corporate proxy or network gateway.
Works with both Argo Server mode and Kubernetes API mode.
If `--proxy-url` is not specified, the CLI will respect the standard `HTTP_PROXY` and `HTTPS_PROXY` environment variables.
