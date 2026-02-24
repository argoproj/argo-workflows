Description: Extend client TLS certificate support in server mode
Authors: [Miltiadis Alexis](https://github.com/miltalex)
Component: CLI
Issues: 13437

The client TLS certificate support functionality has been extended in server mode.
Previously, the `--client-certificate` and `--client-key` flags were inherited from kubectl
and were only used when connecting to Kubernetes directly via client-go (the Kubernetes Go SDK).
The CLI did not use these flags itself and passed them through to client-go. However, in server mode, client-go is not used.

These flags are now reused in server mode by passing them to the gRPC and HTTP clients as well.
This provides consistent client certificate authentication capabilities across different connection modes,
improving the overall flexibility and security of the system.
