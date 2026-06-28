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

The image is published to **`ghcr.io/yunyuyuan/anypaste`**. No secrets to
configure — just give it a volume:

### docker run

```bash
docker run -d --name anypaste \
  -p 8080:8080 \
  -v anypaste-data:/data \
  ghcr.io/yunyuyuan/anypaste:latest
```

### docker compose

```yaml
services:
  anypaste:
    image: ghcr.io/yunyuyuan/anypaste:latest
    ports:
      - "8080:8080"
    volumes:
      - anypaste-data:/data
    restart: unless-stopped

volumes:
  anypaste-data:
```

### First run

Open <http://localhost:8080>. On first launch the app shows a **setup page** —
choose an admin password and you're in. Everything is created automatically:

- the **JWT secret** is generated and saved to `config.json`;
- the **admin password** (bcrypt-hashed) is saved there after you set it;
- the **SQLite DB** is created and migrated.

All of it lives under `/data`, so mount that as a volume to persist across
upgrades. There are no required environment variables.

## Configuration

Everything persists under `/data` (config, DB, uploads). The optional env vars
only change paths/binding:

| Variable      | Default            | Notes                                   |
| ------------- | ------------------ | --------------------------------------- |
| `ADDR`        | `:8080`            | Listen address.                         |
| `DB_PATH`     | `/data/data.db`    | SQLite file location.                   |
| `UPLOAD_DIR`  | `/data/uploads`    | Uploaded files directory.               |
| `CONFIG_PATH` | `/data/config.json`| JWT secret + admin password hash.       |

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
# backend (writes config.json + data.db in the working dir; first run prompts
# for an admin password via the web UI)
go run ./cmd/server

# frontend (Vite dev server, proxies /api to the backend)
cd web && pnpm install && pnpm dev
```
