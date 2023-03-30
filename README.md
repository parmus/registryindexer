Registryindexer
================
![GitHub](https://img.shields.io/github/license/parmus/registryindexer)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/parmus/registryindexer)
![GitHub issues](https://img.shields.io/github/issues-raw/parmus/registryindexer)
![Swagger Validator](https://img.shields.io/swagger/valid/3.0?specUrl=https%3A%2F%2Fraw.githubusercontent.com%2Fparmus%2Fregistryindexer%2Fmaster%2Finternal%2Fapi%2Fdocs%2Fopenapi.json)

Registry Indexer is a small in-memory index for the
[Docker Registry service](https://docs.docker.com/registry/).

This small service exists because the official
[Docker Registry service](https://docs.docker.com/registry/) doesn't
have any search capabilities.


## API
The API is [fully documented](openapi.json) in an
[OpenAPI](https://github.com/OAI/OpenAPI-Specification) specification.
Registryindexer self-host the API specification on `/openapi.json`.
Registryindexer also comes with an embedded [Swagger UI](https://github.com/swagger-api/swagger-ui)
hosted on `/`.

Registryindexer expose Prometheus metrics on the `/metrics` endpoint.

## Local developement
Registryindexer requires Go 1.18

### How to build locally

- `go mod download`
- `go build -v ./cmd/registryindexer`

### How to build the container image
`docker build -t registryindexer .`
