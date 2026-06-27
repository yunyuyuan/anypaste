# anypaste

A tiny, self-hosted paste & file-sharing service. One small Docker image runs
everything: a Go server that serves both the web UI and the API, plus a
cross-platform CLI for the terminal.

## Features

- **Text pastes & file uploads** — create a note or upload any file; large
  uploads stream with progress and resumable downloads (HTTP Range).
- **Edit** — change a paste's content from the web UI.
- **Manage** — list, view, copy, download, and delete from a clean web UI.
- **Single-password auth** — log in with one admin password; the session is a
  JWT kept in a cookie.
- **CLI** — upload / list / download from the terminal. A built-in guide at
  `/help` links to per-platform binaries.
- **Single static image** — the frontend is embedded in the Go binary (pure-Go
  SQLite, no CGO). Multi-arch images for `linux/amd64` and `linux/arm64`.

## Quick start

The image is published to **`ghcr.io/yunyuyuan/anypaste`**.

Two secrets are required:

- `APP_PASSWD` — a **bcrypt hash** of your login password (not plaintext).
- `JWT_SECRET` — a long random string used to sign sessions (keep it stable).

Generate the password hash with the bundled tool (edit the literal inside
`cmd/password/main.go`, then run it):

```bash
go run ./cmd/password    # prints the bcrypt hash
```

### docker run

```bash
docker run -d --name anypaste \
  -p 8080:8080 \
  -e JWT_SECRET="$(openssl rand -base64 32)" \
  -e APP_PASSWD='<bcrypt-hash>' \
  -v anypaste-data:/data \
  ghcr.io/yunyuyuan/anypaste:latest
```

Open <http://localhost:8080> and log in. A fresh setup needs **no database
file** — the server creates and migrates the SQLite DB on first boot.

### docker compose

```yaml
services:
  anypaste:
    image: ghcr.io/yunyuyuan/anypaste:latest
    ports:
      - "8080:8080"
    environment:
      JWT_SECRET: ${JWT_SECRET:?set JWT_SECRET in .env}
      APP_PASSWD: ${APP_PASSWD:?set APP_PASSWD in .env (bcrypt hash)}
    volumes:
      - anypaste-data:/data
    restart: unless-stopped

volumes:
  anypaste-data:
```

## Configuration

| Variable     | Required | Default         | Notes                              |
| ------------ | -------- | --------------- | ---------------------------------- |
| `JWT_SECRET` | **yes**  | _(empty)_       | Random secret for signing JWTs.    |
| `APP_PASSWD` | **yes**  | _(empty)_       | bcrypt hash of the login password. |
| `ADDR`       | no       | `:8080`         | Listen address.                    |
| `DB_PATH`    | no       | `/data/data.db` | SQLite file location.              |
| `UPLOAD_DIR` | no       | `/data/uploads` | Uploaded files directory.          |

`/data` holds the database and uploads — mount it as a volume to persist data.

> **Persistence note:** the container runs rootless (uid `65532`). Named volumes
> (as above) work as-is. For a **host bind mount**, `chown 65532:65532` the host
> directory first, or the server can't write to it.

> **TLS:** the container speaks plain HTTP. Put it behind a reverse proxy
> (Caddy / nginx / Traefik) for HTTPS — the app is same-origin, so just proxy
> `/` to the container.

## CLI

After logging into the web UI, open **`/help`** to download the CLI for your
platform and see the full command reference.

```bash
anypaste login --server https://your-host/api
anypaste up -m "a quick note"
anypaste up ./report.pdf
anypaste ls
anypaste down <id> -o ./report.pdf
```

## Releasing

Pushing a version tag builds and publishes the multi-arch image to GHCR
([`.github/workflows/release.yml`](.github/workflows/release.yml)) — nothing
runs on ordinary pushes:

```bash
git tag v1.0.0
git push origin v1.0.0
```

## Development

```bash
# backend (loads .env.local for JWT_SECRET / APP_PASSWD)
go run ./cmd/server

# frontend (Vite dev server, proxies /api to the backend)
cd web && pnpm install && pnpm dev
```
