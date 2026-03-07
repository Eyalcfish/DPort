#!/usr/bin/env bash
set -e
echo "=== Running DPort benchmarks ==="
CGO_ENABLED=0 go test -run=xxx -bench=Benchmark -benchmem ./src/benchmark/
