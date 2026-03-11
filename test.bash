#!/usr/bin/env bash
set -e
echo "=== Running DPort tests ==="
CGO_ENABLED=0 go test -v ./test/
