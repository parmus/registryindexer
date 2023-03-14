Registry Indexer
================
Registry Indexer is a small in-memory index for the
[Docker Registry service](https://docs.docker.com/registry/).

This small service exists because the official
[Docker Registry service](https://docs.docker.com/registry/) doesn't
have any search capabilities.

## Build
`registryindexer` requires Go 1.18

- `go mod download`
- `go build -v ./cmd/registryindexer`

## Docker image
### Build
`docker build -t registryindexer .`

## API
The API is [fully documented](openapi.json) in an
[OpenAPI](https://github.com/OAI/OpenAPI-Specification) specification.
`registryindexer` self-host the API specification on `/openapi.json`.
`registryindexer` also comes with an embedded [Swagger UI](https://github.com/swagger-api/swagger-ui)
hosted on `/`.

`registryindexer` expose Prometheus metrics on the `/metrics` endpoint.
