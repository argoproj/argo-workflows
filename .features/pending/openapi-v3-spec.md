Component: Build and Development
Issues: 12851
Description: Add OpenAPI v3 spec generation from the existing swagger.json (OpenAPI v2) using getkin/kin-openapi.
Author: [Nancy](https://github.com/nancysangani)

### OpenAPI v3 Spec Generation

Adds `api/openapi-spec/openapi.yaml` (OpenAPI v3.0.3) generated from the existing
`api/openapi-spec/swagger.json` via `hack/api/openapi3/main.go`, wired into `make swagger`.
First step toward migrating away from go-swagger which has no plans to support OpenAPI v3.
