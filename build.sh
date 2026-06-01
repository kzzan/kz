#!/bin/bash

set -euo pipefail

VERSION="${VERSION:-v1.0.0}"
BINARY_NAME="kz"
OUTPUT_DIR="./dist"
LDFLAGS="-s -w -X github.com/kzzan/kz/pkg/cli.version=${VERSION}"

PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
)

mkdir -p "$OUTPUT_DIR"

for platform in "${PLATFORMS[@]}"; do
    GOOS=$(echo "$platform" | cut -d'/' -f1)
    GOARCH=$(echo "$platform" | cut -d'/' -f2)
    
    BINARY_PATH="$OUTPUT_DIR/${BINARY_NAME}-${VERSION}-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        BINARY_PATH="${BINARY_PATH}.exe"
    fi
    
    echo "构建 $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH go build -trimpath -ldflags="$LDFLAGS" -o "$BINARY_PATH" ./
    echo "✓ 构建成功: $BINARY_PATH"
done

echo "所有平台构建完成！文件位置: $OUTPUT_DIR"
