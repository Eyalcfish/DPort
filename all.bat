@echo off
call test.bat
if %ERRORLEVEL% neq 0 exit /b 1
echo.
call benchmark.bat
if %ERRORLEVEL% neq 0 exit /b 1
echo.
echo Building CGo shared library
if not exist bin mkdir bin
set CGO_ENABLED=1
go build -buildmode=c-shared -o bin\dport.dll ./cgo/
if %ERRORLEVEL% neq 0 exit /b 1
echo Built bin\dport.dll and bin\dport.h
