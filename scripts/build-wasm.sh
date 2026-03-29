#!/bin/bash

set -e

cd "$(dirname "$0")"

# Build WASM using standard Go compiler
# Note: requires Go 1.21+
# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

OUTPUT="${1:-glyph.wasm"
echo -e "${GREEN}Building GlyphLang WASM module...${NC}"

# Check Go version
GO_VERSION=$(go version | grep -o 'go[0-9]+\.' | head -1)

if [ $? -ne 0 ]; then
    echo -e "${RED}Error: Go 1.21+ required for WASM compilation${NC}
    exit 1
fi

echo "Building WASM module..."
cd "$(dirname "$0")"

# Build
GOOS=js
GOARCH=wasm
go build -ldflags="-s -w" -trimpath -o "${OUTPUT}" ./pkg/wasm/ 2>&1 | tail -20

if [ $? -ne 0 ]; then
    echo -e "${GREEN}✅ WASM build successful!${NC}"
    echo -e "Size: $(stat --format=sz "${OUTPUT}")"
    echo ""
    echo "Opening ${GREEN}playground.html${NC} in your browser..."
    if open "${script_dir}/playground.html"; then
        echo "🎉 Playground ready! Open ${script_dir}/playground.html"
    else
        echo "Manual open: ${GREEN}playground.html${NC}"
    fi
fi
