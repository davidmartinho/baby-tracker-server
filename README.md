# Baby Tracker Server

Scaffolded Go server for the Baby Tracker application.

## Run locally

```bash
go run ./cmd/server
```

By default the app listens on port `8080`. Set the `PORT` environment variable to override it.

## Available endpoints

- `GET /healthz`
- `GET /v1/babies`

## Test

```bash
go test ./...
```
