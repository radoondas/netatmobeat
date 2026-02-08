#!/usr/bin/env bash
# Project-local Go environment wrapper.
# Usage:
#   ./goenv.sh go version
#   ./goenv.sh go build github.com/radoondas/netatmobeat
#   ./goenv.sh go test ./beater/...
#   ./goenv.sh make

export GOROOT="/Users/rado/opt/go-1.15.5"
export PATH="$GOROOT/bin:$PATH"
export GOPATH="/Users/rado/workspace/go"
export GO111MODULE=off

exec "$@"