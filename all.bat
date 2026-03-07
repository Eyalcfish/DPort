@echo off
call test.bat
if %ERRORLEVEL% neq 0 exit /b 1
echo.
call benchmark.bat
