#!/bin/sh
# generates payloads for each os

build() {
GOOS=windows go build -o ../payloads/sandcat.go-windows -ldflags="-s -w" sandcat.go
GOOS=linux go build -o ../payloads/sandcat.go-linux -ldflags="-s -w" sandcat.go
GOOS=darwin go build -o ../payloads/sandcat.go-darwin -ldflags="-s -w" sandcat.go
GOOS=freebsd go build -o ../payloads/sandcat.go-freebsd -ldflags="-s -w" sandcat.go
}
cd gocat && build
cd ..
