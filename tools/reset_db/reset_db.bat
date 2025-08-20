@echo off
setlocal enabledelayedexpansion
REM change to this script's directory regardless of caller's CWD
cd /d %~dp0
echo ========================================
echo        IM System Database Reset Tool
echo ========================================
echo.

echo Compiling database reset tool...
go build -o reset_db.exe .

if %errorlevel% neq 0 (
    echo Compilation failed!
    pause
    exit /b 1
)

echo Compilation successful!
echo.
echo Running database reset tool...
echo.

reset_db.exe

echo.
echo Press any key to exit...
pause > nul


