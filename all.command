#!/usr/bin/env bash
set -e

# Change to the directory where the script is located
cd "$(dirname "$0")"

echo "Running DPort tests"
bash test.bash

echo ""
bash benchmark.bash

echo ""
echo "Building CGo shared library for macOS"
mkdir -p bin
CGO_ENABLED=1 go build -buildmode=c-shared -o bin/dport.dylib ./cgo/
echo "Built bin/dport.dylib and bin/dport.h"
