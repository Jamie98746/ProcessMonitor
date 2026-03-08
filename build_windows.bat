@echo off
rem Build script for Windows (cmd)
rem Usage:
rem   build_windows.bat            # build for current host platform
rem   build_windows.bat linux amd64  # cross-compile to linux/amd64
rem   set "GOOS=darwin"& set "GOARCH=amd64"& build_windows.bat  # via env

setlocal
set "BUILD_DIR=build"
set "BINARY_NAME=procmon"
set "MAIN_PATH=./cmd"

if not exist "%BUILD_DIR%" mkdir "%BUILD_DIR%"

rem Allow overriding via args first, then env vars, then go env defaults
if not "%~1"=="" set "GOOS=%~1"
if not "%~2"=="" set "GOARCH=%~2"

if "%GOOS%"=="" (
    for /f "delims=" %%a in ('go env GOOS') do set "GOOS=%%a"
)

if "%GOARCH%"=="" (
    for /f "delims=" %%a in ('go env GOARCH') do set "GOARCH=%%a"
)

if /I "%GOOS%"=="windows" (
    set "EXE=.exe"
) else (
    set "EXE="
)

rem Strip debug symbols to reduce binary size (can be overridden by setting LDFLAGS)
rem Use one of these to override (in cmd.exe):
rem   set "LDFLAGS=-s -w"    (default, strips symbols)
rem   set "LDFLAGS=NONE"     (disables -ldflags entirely)
rem   set "LDFLAGS=..."      (custom flags)
rem
rem If you run this from PowerShell, use:
rem   $env:LDFLAGS = "NONE"; .\build_windows.bat
rem or:
rem   cmd /c "set \"LDFLAGS=NONE\" && build_windows.bat"
if not defined LDFLAGS (
    set "LDFLAGS=-s -w"
) else if /I "%LDFLAGS%"=="NONE" (
    set "LDFLAGS="
)

echo Building %GOOS%/%GOARCH%...
echo LDFLAGS=%LDFLAGS%
go build -ldflags="%LDFLAGS%" -o "%BUILD_DIR%\%BINARY_NAME%_%GOOS%_%GOARCH%%EXE%" %MAIN_PATH%

endlocal
