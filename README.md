# Baby Tracker Server

Scaffolded Go server for the Baby Tracker application.

## Run locally

```bash
go run ./cmd/server
```

By default the app listens on port `8080`. Set the `PORT` environment variable to override it.

## Deploy to Fly.io

This repository includes a `fly.toml` and `Dockerfile` for Fly.io deployments.

1. Install and authenticate the Fly CLI.
2. Create the Fly app once:
   ```bash
   fly launch --no-deploy --copy-config
   ```
3. Deploy:
   ```bash
   fly deploy
   ```
4. Verify health check:
   ```bash
   fly checks list
   ```

The Fly service is configured to route HTTP traffic to internal port `8080` and run health checks on `GET /healthz`.

## Available endpoints

- `GET /healthz`
- `GET /v1/babies`
- `GET /v1/profile`

### Health check response

`GET /healthz` returns `200 OK` with:

```json
{"status":"ok"}
```

## Test

```bash
go test ./...
```
