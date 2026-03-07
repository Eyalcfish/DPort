@echo off
echo === Running DPort benchmarks ===
set CGO_ENABLED=0
go test -run=xxx -bench=Benchmark -benchmem .\src\benchmark\
