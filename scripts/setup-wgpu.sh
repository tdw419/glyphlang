#!/bin/bash
# Download and install wgpu-native for GlyphLang GPU execution

set -e

WGPU_VERSION="0.19.4.1"
WGPU_DIR="$(pwd)/wgpu-native"
WGPU_URL="https://github.com/gfx-rs/wgpu-native/releases/download/v${WGPU_VERSION}/wgpu-linux-x86_64-release.zip"

echo "=== Installing wgpu-native ${WGPU_VERSION} ==="

# Create directory
mkdir -p "${WGPU_DIR}"
cd "${WGPU_DIR}"

# Download
if [ ! -f "wgpu-linux-x86_64-release.zip" ]; then
    echo "Downloading wgpu-native..."
    curl -L -o wgpu-linux-x86_64-release.zip "${WGPU_URL}"
fi

# Extract
echo "Extracting..."
unzip -o wgpu-linux-x86_64-release.zip

# Setup symlinks
echo "Setting up library paths..."
mkdir -p ../pkg/gpu/native
cp -f libwgpu_native.so ../pkg/gpu/native/
cp -rf include ../pkg/gpu/native/

echo ""
echo "✅ wgpu-native installed to pkg/gpu/native/"
echo "   Library: pkg/gpu/native/libwgpu_native.so"
echo "   Headers: pkg/gpu/native/include/"
echo ""
echo "Run 'go build -tags gpu ./cmd/glyph' to build with GPU support"
