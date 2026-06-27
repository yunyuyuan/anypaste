#!/usr/bin/env bash
# Cross-compile the anypaste CLI for all supported platforms into ./cli-dist.
# These binaries are served by the server at /cli/ and linked from the /help page.
set -euo pipefail

cd "$(dirname "$0")/.."

OUT=cli-dist
PKG=./cmd/cli
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
  name="anypaste-${os}-${arch}"
  [ "$os" = "windows" ] && name="${name}.exe"
  echo "building ${name}"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
    go build -trimpath -ldflags "-s -w" -o "$OUT/${name}" "$PKG"
done

echo "done -> ${OUT}/"
ls -lh "$OUT"
