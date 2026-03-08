#!/usr/bin/env bash
# Build script for Unix-like platforms (Linux/macOS)
# Usage:
#   chmod +x build_unix.sh
#   ./build_unix.sh                # build for current platform
#   GOOS=linux GOARCH=amd64 ./build_unix.sh  # cross-compile

set -eu

BUILD_DIR=build
BINARY_NAME=procmon
MAIN_PATH=./cmd

mkdir -p "$BUILD_DIR"

# Allow overriding GOOS/GOARCH from the environment.
GOOS=${GOOS:-$(go env GOOS)}
GOARCH=${GOARCH:-$(go env GOARCH)}

echo "Building $GOOS/$GOARCH..."

# Strip debug symbols to reduce binary size (aligns with Docker build settings).
# Can be overridden by setting LDFLAGS.
# - default: LDFLAGS='-s -w'
# - to disable:           export LDFLAGS=NONE
# - to keep empty:        export LDFLAGS=""
if [ -z "${LDFLAGS+set}" ]; then
  LDFLAGS='-s -w'
elif [ "$LDFLAGS" = "NONE" ]; then
  LDFLAGS=''
fi

echo "Building $GOOS/$GOARCH..."
echo "LDFLAGS=$LDFLAGS"

go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/${BINARY_NAME}_${GOOS}_${GOARCH}" "$MAIN_PATH"
