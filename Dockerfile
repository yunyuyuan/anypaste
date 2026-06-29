# syntax=docker/dockerfile:1
#
# Minimal runtime image. The Go server is cross-compiled on the CI host (see
# .github/workflows/release.yml, `build` job) — that produces server-dist/
# binaries with the frontend and the CLI downloads already embedded. This
# Dockerfile just packages the right one into a distroless image.
#
# Build context expects:  server-dist/anypaste-server-linux-<arch>
# Use docker buildx with --platform linux/amd64,linux/arm64; TARGETARCH picks
# the right pre-built binary per arch.

# Tiny prep stage just to create /data with the right ownership for the
# distroless nonroot user (distroless has no shell / chown).
FROM --platform=$BUILDPLATFORM alpine:3 AS prep
RUN mkdir -p /data

FROM gcr.io/distroless/static-debian12:nonroot
ARG TARGETARCH
WORKDIR /app
COPY server-dist/anypaste-server-linux-${TARGETARCH} /app/anypaste
COPY --from=prep --chown=65532:65532 /data /data
ENV ADDR=:8080 \
    DB_PATH=/data/data.db \
    UPLOAD_DIR=/data/uploads \
    CONFIG_PATH=/data/config.json
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/app/anypaste"]
