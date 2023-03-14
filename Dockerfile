## Builder image
FROM golang:1.18.3-alpine AS builder
WORKDIR /builddir
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v ./cmd/registryindexer

## Deployment image
FROM scratch AS registryindexer
EXPOSE 5010
VOLUME ["/cache"]
COPY --from=builder /etc/ssl /etc/ssl
COPY --from=builder /builddir/registryindexer /registryindexer
ENTRYPOINT ["/registryindexer"]
