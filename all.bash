#!/usr/bin/env bash
set -e
bash test.bash
echo ""
bash benchmark.bash
echo ""
echo "Building CGo shared library"
mkdir -p bin
go build -buildmode=c-shared -o bin/dport.so ./cgo/
echo "Built bin/dport.so and bin/dport.h"
