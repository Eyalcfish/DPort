#!/usr/bin/env bash
set -e
echo "Running DPort benchmarks..."
mkdir -p benchmark_logs
LOG_FILE="benchmark_logs/bench_$(date +%Y-%m-%d_%H-%M-%S).log"

echo "Saving results to $LOG_FILE..."
CGO_ENABLED=0 go test -run=xxx -bench=Benchmark -benchmem ./benchmark/ | tee "$LOG_FILE"
