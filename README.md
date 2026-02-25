# Baby Tracker Server

Scaffolded Go server for the Baby Tracker application.

## Run locally

Start Postgres first (example with Docker):

```bash
docker run --name baby-tracker-postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=baby_tracker \
  -p 5432:5432 \
  -d postgres:16
```

Set `DATABASE_URL`, then run the server:

```bash
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/baby_tracker?sslmode=disable
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
- `POST /v1/babies/{babyID}/events`
- `GET /v1/profile`

### Health check response

`GET /healthz` returns `200 OK` with:

```json
{"status":"ok"}
```

## Test

```bash
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/baby_tracker?sslmode=disable
go test ./...
```
