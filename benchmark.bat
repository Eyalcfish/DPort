@echo off
echo === Running DPort benchmarks ===
set CGO_ENABLED=0

if not exist benchmark_logs mkdir benchmark_logs
for /f "tokens=2-4 delims=/ " %%a in ('date /t') do (set mydate=%%c-%%a-%%b)
for /f "tokens=1-2 delims=/:" %%a in ('time /t') do (set mytime=%%a%%b)
set LOG_FILE=benchmark_logs\bench_%mydate%_%mytime%.log
set LOG_FILE=%LOG_FILE: =_%

echo Saving results to "%LOG_FILE%"...
go test -run=xxx -bench=Benchmark -benchmem .\src\benchmark\ > "%LOG_FILE%"
type "%LOG_FILE%"
