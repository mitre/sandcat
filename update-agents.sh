#!/bin/bash
# generates payloads for each os
GOOS=windows go build -o payloads/sandcat.go-windows -ldflags="-s -w" gocat/sandcat.go
GOOS=linux go build -o payloads/sandcat.go-linux -ldflags="-s -w" gocat/sandcat.go
GOOS=darwin go build -o payloads/sandcat.go-darwin -ldflags="-s -w" gocat/sandcat.go
