@echo off
echo === Running DPort tests ===
set CGO_ENABLED=0
go test -v .\src\test\
if %ERRORLEVEL% neq 0 (
    echo TESTS FAILED
    exit /b 1
)
