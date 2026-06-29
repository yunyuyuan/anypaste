#!/usr/bin/env bash
# Cross-compile the anypaste server for all supported platforms into ./server-dist.
# The frontend must already be embedded at internal/web/dist (run `pnpm build` in
# web/ and copy its dist/ there first — the Dockerfile and release workflow do this).
set -euo pipefail

cd "$(dirname "$0")/.."

OUT=server-dist
PKG=./cmd/server
mkdir -p "$OUT"

platforms=(
  "windows/amd64"
  "windows/arm64"
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
)

for p in "${platforms[@]}"; do
  os="${p%/*}"
  arch="${p#*/}"
  name="anypaste-server-${os}-${arch}"
  [ "$os" = "windows" ] && name="${name}.exe"
  echo "building ${name}"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
    go build -trimpath -ldflags "-s -w" -o "$OUT/${name}" "$PKG"
done

echo "done -> ${OUT}/"
ls -lh "$OUT"
