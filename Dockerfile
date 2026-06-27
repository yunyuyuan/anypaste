# syntax=docker/dockerfile:1

# 1) Build the frontend. JS output is arch-independent, so this always runs on
#    the build host (fast, no emulation).
FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend
WORKDIR /app/web
RUN corepack enable && corepack prepare pnpm@10 --activate
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

# 2) Build the Go server, cross-compiled to the target arch, with the frontend
#    embedded. Pure-Go sqlite means CGO can stay off → fully static binary.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS backend
WORKDIR /src
RUN apk add --no-cache bash
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Embed the built frontend, then build the multi-platform CLI binaries that the
# server hands out at /cli (linked from the /help page).
COPY --from=frontend /app/web/dist ./internal/web/dist
RUN bash scripts/build-cli.sh
# A pre-owned data dir so the named volume is writable by the nonroot user.
RUN mkdir -p /data
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags "-s -w" -o /out/anypaste ./cmd/server

# 3) Minimal, rootless runtime image.
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=backend /out/anypaste /app/anypaste
COPY --from=backend /src/cli-dist /app/cli-dist
COPY --from=backend --chown=65532:65532 /data /data
ENV ADDR=:8080 \
    DB_PATH=/data/data.db \
    UPLOAD_DIR=/data/uploads
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/app/anypaste"]
