#!/bin/bash
# Generates sandcat payloads for each OS, using all Go proxy extensions from gocat-extensions/proxy

set -e

BUILD_DIR="../tmp_gocat_build"
GOCAT_DIR="."
EXTENSIONS_DIR="../gocat-extensions/proxy"

function build() {
    echo "[*] Creating temp build directory: ${BUILD_DIR}"
    
    mkdir -p "${BUILD_DIR}"
    mkdir -p "${BUILD_DIR}/gocat"
    mkdir -p "${BUILD_DIR}/gocat-extensions"

    echo "[*] Copying all gocat source files..."
    cp -r ${GOCAT_DIR}/* "${BUILD_DIR}/gocat"

    echo "[*] Copying all gocat-extensions source files..."
    cp -r ${GOCAT_DIR}/* "${BUILD_DIR}/gocat-extensions"

    echo "[*] Copying all proxy extension files from ${EXTENSIONS_DIR}..."
    if compgen -G "${EXTENSIONS_DIR}/*.go" > /dev/null; then
        cp "${EXTENSIONS_DIR}"/*.go "${BUILD_DIR}/gocat/proxy/"
    else
        echo "[!] No .go files found in ${EXTENSIONS_DIR}"
        exit 1
    fi
    cd "${BUILD_DIR}/gocat"
    echo "[*] Building sandcat payloads from ${BUILD_DIR}..."
    GOOS=windows GOARCH=amd64 go build -o ../../payloads/sandcat.go-windows -ldflags="-s -w" sandcat.go
    GOOS=linux   GOARCH=amd64 go build -o ../../payloads/sandcat.go-linux   -ldflags="-s -w" sandcat.go
    GOOS=darwin  GOARCH=amd64 go build -o ../../payloads/sandcat.go-darwin  -ldflags="-s -w" sandcat.go
    GOOS=darwin  GOARCH=arm64 go build -o ../../payloads/sandcat.go-darwin-arm64 -ldflags="-s -w" sandcat.go


    echo "[*] Cleaning up temporary build directory..."
    
    cd ../ && rm -rf "${BUILD_DIR}"
}
cd gocat
build
cd ..