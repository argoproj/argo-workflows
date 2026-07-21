Description: Extend client TLS certificate support in server mode
Authors: [Miltiadis Alexis](https://github.com/miltalex)
Component: CLI
Issues: 13437

Use `--client-certificate` and `--client-key` when an Argo Server or its proxy requires mutual TLS authentication.
Both flags must be provided together.
The certificate is used by the gRPC and HTTP/1 clients, including artifact downloads with `argo cp`.

For example, run `argo --argo-server argo.example.com:443 --secure --client-certificate client.crt --client-key client.key list`.

In server mode, client certificates embedded in a kubeconfig context are not used automatically.
Pass the two flags explicitly when connecting through Argo Server.
The flags do not have `ARGO_*` environment variable equivalents.
