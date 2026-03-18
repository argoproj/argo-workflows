Description: Bearer Token Authentication for HTTP Artifacts
Authors: [Fredrik Karlström](https://github.com/lfk)
Component: Artifacts
Issues: 15778

HTTP input and output artifacts can now authenticate using a Kubernetes Secret-backed bearer token. Set `auth.bearer.tokenSecret` on any `http` artifact to have the executor inject an `Authorization: Bearer <token>` header automatically.
