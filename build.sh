#!/bin/sh
# Baut die Linux-Programme nach dist/ (amd64 + arm64).
set -e
cd "$(dirname "$0")"
VER="$(grep -o 'version = "[^"]*"' panel/main.go | cut -d'"' -f2)"
mkdir -p dist
for a in amd64 arm64; do
  (cd panel && CGO_ENABLED=0 GOOS=linux GOARCH=$a go build -trimpath -ldflags "-s -w" -o ../dist/serverwerk-linux-$a .)
done
# Immer Unix-Zeilenenden, sonst bricht sh auf dem Server ab.
tr -d '\r' < install.sh > dist/install.sh
chmod 755 dist/install.sh
echo "$VER" > dist/version.txt
ls -la dist
